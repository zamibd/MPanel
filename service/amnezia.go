package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zamibd/MPanel/database"
	"github.com/zamibd/MPanel/database/model"
	"github.com/zamibd/MPanel/logger"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

// AmneziaConfig holds the server-level configuration for the AmneziaWG interface.
type AmneziaConfig struct {
	Interface  string `json:"interface"`  // e.g. "awg0"
	PrivateKey string `json:"privateKey"` // Server private key (Base64)
	PublicKey  string `json:"publicKey"`  // Server public key (Base64, derived)
	Address    string `json:"address"`    // Server tunnel IP e.g. "10.8.0.1/24"
	ListenPort int    `json:"listenPort"` // UDP port, e.g. 51820
	DNS        string `json:"dns"`        // e.g. "1.1.1.1"
	PostUp     string `json:"postUp"`     // iptables rules
	PostDown   string `json:"postDown"`   // iptables teardown rules
}

// AmneziaService manages a standalone WireGuard/AmneziaWG VPN interface
// independently of the Sing-box core.
type AmneziaService struct {
	mu      sync.RWMutex
	running bool
	cfg     *AmneziaConfig
	ss      SettingService
}

var amneziaServiceInstance = &AmneziaService{}

// GetAmneziaService returns the singleton AmneziaService.
func GetAmneziaService() *AmneziaService {
	return amneziaServiceInstance
}

// ------------------------------------------------------------------
// Key Generation
// ------------------------------------------------------------------

// GenerateKeypair generates a fresh WireGuard private/public keypair.
func (s *AmneziaService) GenerateKeypair() (privateKey, publicKey string, err error) {
	pk, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("generate wireguard key: %w", err)
	}
	privateKey = base64.StdEncoding.EncodeToString(pk[:])
	pub := pk.PublicKey()
	publicKey = base64.StdEncoding.EncodeToString(pub[:])
	return
}

// DerivePublicKey derives the public key from a Base64-encoded private key.
func (s *AmneziaService) DerivePublicKey(privateKeyB64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}
	pk, err := wgtypes.NewKey(raw)
	if err != nil {
		return "", fmt.Errorf("parse wireguard key: %w", err)
	}
	pub := pk.PublicKey()
	return base64.StdEncoding.EncodeToString(pub[:]), nil
}

// ------------------------------------------------------------------
// Server Config (stored in DB Settings table under key "amneziaConfig")
// ------------------------------------------------------------------

// GetConfig loads the AmneziaWG server config from the database.
func (s *AmneziaService) GetConfig() (*AmneziaConfig, error) {
	val, err := s.ss.getString("amneziaConfig")
	if err != nil {
		// Not found → not configured yet
		return nil, nil
	}
	if val == "" {
		return nil, nil
	}
	cfg := &AmneziaConfig{}
	if err := json.Unmarshal([]byte(val), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveConfig persists the AmneziaWG server config to the database.
func (s *AmneziaService) SaveConfig(cfg *AmneziaConfig) error {
	// Auto-derive public key from private key
	if cfg.PrivateKey != "" && cfg.PublicKey == "" {
		pub, err := s.DerivePublicKey(cfg.PrivateKey)
		if err == nil {
			cfg.PublicKey = pub
		}
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.ss.saveSetting("amneziaConfig", string(raw))
}

// ------------------------------------------------------------------
// Peer CRUD
// ------------------------------------------------------------------

// GetAllPeers returns all AmneziaPeer records.
func (s *AmneziaService) GetAllPeers() ([]model.AmneziaPeer, error) {
	db := database.GetDB()
	var peers []model.AmneziaPeer
	err := db.Order("id asc").Find(&peers).Error
	return peers, err
}

// GetPeer returns a single peer by ID.
func (s *AmneziaService) GetPeer(id uint) (*model.AmneziaPeer, error) {
	db := database.GetDB()
	var peer model.AmneziaPeer
	err := db.First(&peer, id).Error
	return &peer, err
}

// AddPeer creates a new peer (auto-generates keys/IP if needed) and live-applies it.
func (s *AmneziaService) AddPeer(peer *model.AmneziaPeer) error {
	// Auto-generate keypair if private key is missing or "auto"
	if peer.PrivateKey == "" || peer.PrivateKey == "auto" {
		priv, pub, err := s.GenerateKeypair()
		if err != nil {
			return err
		}
		peer.PrivateKey = priv
		peer.PublicKey = pub
	} else if peer.PublicKey == "" {
		pub, err := s.DerivePublicKey(peer.PrivateKey)
		if err != nil {
			return err
		}
		peer.PublicKey = pub
	}

	// Auto-assign the next available tunnel IP
	if peer.AllowedIPs == "" || peer.AllowedIPs == "auto" {
		ip, err := s.nextAvailableIP()
		if err != nil {
			return err
		}
		peer.AllowedIPs = ip + "/32"
	}

	db := database.GetDB()
	if err := db.Create(peer).Error; err != nil {
		return err
	}

	// Live-apply to running interface
	if s.IsRunning() {
		if err := s.applyPeerToInterface(peer); err != nil {
			logger.Warning("amnezia: failed to apply peer to interface: ", err)
		}
	}
	return nil
}

// EditPeer updates an existing peer and re-applies to the interface.
func (s *AmneziaService) EditPeer(peer *model.AmneziaPeer) error {
	db := database.GetDB()
	if err := db.Save(peer).Error; err != nil {
		return err
	}
	if s.IsRunning() {
		if err := s.applyPeerToInterface(peer); err != nil {
			logger.Warning("amnezia: failed to re-apply peer: ", err)
		}
	}
	return nil
}

// DeletePeer removes a peer by ID from the DB and live interface.
func (s *AmneziaService) DeletePeer(id uint) error {
	peer, err := s.GetPeer(id)
	if err != nil {
		return err
	}
	if s.IsRunning() {
		_ = s.removePeerFromInterface(peer.PublicKey)
	}
	db := database.GetDB()
	return db.Delete(&model.AmneziaPeer{}, id).Error
}

// TogglePeer enables or disables a peer without a restart.
func (s *AmneziaService) TogglePeer(id uint) (*model.AmneziaPeer, error) {
	peer, err := s.GetPeer(id)
	if err != nil {
		return nil, err
	}
	peer.Enable = !peer.Enable
	db := database.GetDB()
	if err := db.Save(peer).Error; err != nil {
		return nil, err
	}
	if s.IsRunning() {
		if peer.Enable {
			_ = s.applyPeerToInterface(peer)
		} else {
			_ = s.removePeerFromInterface(peer.PublicKey)
		}
	}
	return peer, nil
}

// ------------------------------------------------------------------
// Interface Management
// ------------------------------------------------------------------

// StartInterface brings up the WireGuard/AmneziaWG tunnel interface.
func (s *AmneziaService) StartInterface() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return fmt.Errorf("amnezia: server config not found, configure it first via /apiv2/amnezia/config")
	}
	s.cfg = cfg

	iface := cfg.Interface
	if iface == "" {
		iface = "awg0"
		cfg.Interface = iface
	}

	confDir := "/etc/amnezia/amneziawg"
	_ = os.MkdirAll(confDir, 0700)
	confPath := filepath.Join(confDir, iface+".conf")

	if err := s.writeWgConf(confPath, cfg); err != nil {
		return fmt.Errorf("amnezia: write config: %w", err)
	}

	// Try awg-quick first (AmneziaWG), fallback to standard wg-quick
	if err := runCmd("awg-quick", "up", confPath); err != nil {
		if err2 := runCmd("wg-quick", "up", confPath); err2 != nil {
			return fmt.Errorf("amnezia: bring up interface (awg-quick err: %v)", err)
		}
	}

	// Live-apply all enabled peers
	peers, _ := s.GetAllPeers()
	for i := range peers {
		if peers[i].Enable {
			_ = s.applyPeerToInterface(&peers[i])
		}
	}

	s.running = true
	logger.Info("AmneziaWG interface ", iface, " started")
	return nil
}

// StopInterface tears down the WireGuard tunnel interface.
func (s *AmneziaService) StopInterface() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, _ := s.GetConfig()
	iface := "awg0"
	if cfg != nil && cfg.Interface != "" {
		iface = cfg.Interface
	}
	confPath := fmt.Sprintf("/etc/amnezia/amneziawg/%s.conf", iface)
	_ = runCmd("awg-quick", "down", confPath)
	_ = runCmd("wg-quick", "down", confPath)
	s.running = false
	logger.Info("AmneziaWG interface ", iface, " stopped")
	return nil
}

func (s *AmneziaService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ------------------------------------------------------------------
// Traffic Stats
// ------------------------------------------------------------------

// SyncTraffic reads peer traffic counters from the wg interface and persists them.
func (s *AmneziaService) SyncTraffic() error {
	if !s.IsRunning() || s.cfg == nil {
		return nil
	}

	iface := s.cfg.Interface
	if iface == "" {
		iface = "awg0"
	}

	out, err := exec.Command("awg", "show", iface, "transfer").Output()
	if err != nil {
		out, err = exec.Command("wg", "show", iface, "transfer").Output()
		if err != nil {
			return nil
		}
	}

	// Output format: <pubkey>\t<rx_bytes>\t<tx_bytes>
	db := database.GetDB()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		pubKey := parts[0]
		var rx, tx int64
		fmt.Sscanf(parts[1], "%d", &rx)
		fmt.Sscanf(parts[2], "%d", &tx)

		db.Model(&model.AmneziaPeer{}).
			Where("public_key = ? AND enable = true", pubKey).
			Updates(map[string]interface{}{
				"down": gorm.Expr("down + ?", rx),
				"up":   gorm.Expr("up + ?", tx),
			})
	}
	return nil
}

// DepletePeers disables peers that have exceeded volume or expiry.
func (s *AmneziaService) DepletePeers() error {
	db := database.GetDB()
	dt := time.Now().Unix()

	var toDisable []model.AmneziaPeer
	err := db.Model(&model.AmneziaPeer{}).
		Where("enable = true AND ((volume > 0 AND up + down > volume) OR (expiry > 0 AND expiry < ?))", dt).
		Find(&toDisable).Error
	if err != nil {
		return err
	}

	for i := range toDisable {
		toDisable[i].Enable = false
		db.Save(&toDisable[i])
		if s.IsRunning() {
			_ = s.removePeerFromInterface(toDisable[i].PublicKey)
		}
		logger.Infof("AmneziaWG: disabled peer '%s' - volume/expiry exceeded", toDisable[i].Name)
	}
	return nil
}

// ------------------------------------------------------------------
// Client Config Generation
// ------------------------------------------------------------------

// GenerateClientConfig generates a ready-to-use WireGuard/AmneziaWG .conf string.
func (s *AmneziaService) GenerateClientConfig(peer *model.AmneziaPeer, serverIP string) (string, error) {
	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return "", fmt.Errorf("server config not found")
	}

	serverPub := cfg.PublicKey
	if serverPub == "" {
		serverPub, err = s.DerivePublicKey(cfg.PrivateKey)
		if err != nil {
			return "", err
		}
	}

	// Convert "10.8.0.2/32" → "10.8.0.2/24" for client Address
	clientAddr := strings.Replace(peer.AllowedIPs, "/32", "/24", 1)

	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", peer.PrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", clientAddr))
	if cfg.DNS != "" {
		sb.WriteString(fmt.Sprintf("DNS = %s\n", cfg.DNS))
	}
	sb.WriteString("\n[Peer]\n")
	sb.WriteString(fmt.Sprintf("PublicKey = %s\n", serverPub))
	sb.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	sb.WriteString(fmt.Sprintf("Endpoint = %s:%d\n", serverIP, cfg.ListenPort))
	sb.WriteString("PersistentKeepalive = 25\n")

	return sb.String(), nil
}

// ------------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------------

func (s *AmneziaService) applyPeerToInterface(peer *model.AmneziaPeer) error {
	iface := s.getIface()
	return runCmd("wg", "set", iface,
		"peer", peer.PublicKey,
		"allowed-ips", peer.AllowedIPs,
	)
}

func (s *AmneziaService) removePeerFromInterface(pubKey string) error {
	iface := s.getIface()
	return runCmd("wg", "set", iface, "peer", pubKey, "remove")
}

func (s *AmneziaService) getIface() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cfg != nil && s.cfg.Interface != "" {
		return s.cfg.Interface
	}
	return "awg0"
}

func (s *AmneziaService) nextAvailableIP() (string, error) {
	peers, err := s.GetAllPeers()
	if err != nil {
		return "", err
	}

	base := "10.8.0"
	cfg, _ := s.GetConfig()
	if cfg != nil && cfg.Address != "" {
		_, ipNet, err := net.ParseCIDR(cfg.Address)
		if err == nil {
			parts := strings.Split(ipNet.IP.String(), ".")
			if len(parts) >= 3 {
				base = strings.Join(parts[:3], ".")
			}
		}
	}

	used := map[string]bool{}
	for _, p := range peers {
		ip := strings.Split(p.AllowedIPs, "/")[0]
		used[ip] = true
	}

	for i := 2; i <= 254; i++ {
		candidate := fmt.Sprintf("%s.%d", base, i)
		if !used[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available IP in range %s.0/24", base)
}

func (s *AmneziaService) writeWgConf(path string, cfg *AmneziaConfig) error {
	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", cfg.PrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", cfg.Address))
	sb.WriteString(fmt.Sprintf("ListenPort = %d\n", cfg.ListenPort))
	if cfg.PostUp != "" {
		sb.WriteString(fmt.Sprintf("PostUp = %s\n", cfg.PostUp))
	}
	if cfg.PostDown != "" {
		sb.WriteString(fmt.Sprintf("PostDown = %s\n", cfg.PostDown))
	}

	peers, _ := s.GetAllPeers()
	for _, p := range peers {
		if !p.Enable {
			continue
		}
		sb.WriteString("\n[Peer]\n")
		sb.WriteString(fmt.Sprintf("# Name: %s\n", p.Name))
		sb.WriteString(fmt.Sprintf("PublicKey = %s\n", p.PublicKey))
		sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", p.AllowedIPs))
	}

	return os.WriteFile(path, []byte(sb.String()), 0600)
}

func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %s (%w)", name, strings.Join(args, " "), string(out), err)
	}
	return nil
}
