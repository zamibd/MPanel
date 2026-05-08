package service

// amnezia.go – Shared types, DB methods and keypair generation for AmneziaWG.
// Platform-specific interface management lives in:
//   - amnezia_linux.go  (//go:build linux)
//   - amnezia_stub.go   (//go:build !linux)

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/zamibd/MPanel/database"
	"github.com/zamibd/MPanel/database/model"
	"github.com/zamibd/MPanel/logger"

	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

// AmneziaConfig holds the server-level configuration for the AmneziaWG interface.
// Persisted as JSON in the Settings table under key "amneziaConfig".
type AmneziaConfig struct {
	Interface  string `json:"interface"`  // e.g. "awg0"
	PrivateKey string `json:"privateKey"` // Base64-std server private key
	PublicKey  string `json:"publicKey"`  // Base64-std server public key (derived)
	Address    string `json:"address"`    // Server tunnel CIDR e.g. "10.8.0.1/24"
	ListenPort int    `json:"listenPort"` // UDP listen port e.g. 51820
	DNS        string `json:"dns"`        // Client DNS, e.g. "1.1.1.1"
	MTU        int    `json:"mtu"`        // 0 → 1420

	// iptables hook commands
	PostUp   string `json:"postUp"`
	PostDown string `json:"postDown"`

	// AmneziaWG obfuscation parameters (optional; omit for standard WireGuard)
	Jc   int `json:"jc,omitempty"`   // Junk packet count
	Jmin int `json:"jmin,omitempty"` // Junk packet min size
	Jmax int `json:"jmax,omitempty"` // Junk packet max size
	S1   int `json:"s1,omitempty"`   // Init packet junk size
	S2   int `json:"s2,omitempty"`   // Response packet junk size
	H1   int `json:"h1,omitempty"`   // Init packet magic header
	H2   int `json:"h2,omitempty"`   // Response packet magic header
	H3   int `json:"h3,omitempty"`   // Cookie packet magic header
	H4   int `json:"h4,omitempty"`   // Transport packet magic header
}

// AmneziaService manages a standalone AmneziaWG VPN tunnel.
// The `deviceHandle` field holds a *awgDevice.Device on Linux (type-asserted
// inside amnezia_linux.go) and is nil on other platforms.
type AmneziaService struct {
	mu           sync.RWMutex
	deviceHandle any // *awgDevice.Device on Linux
	running      bool
	cfg          *AmneziaConfig
	ss           SettingService
}

var amneziaServiceInstance = &AmneziaService{}

// GetAmneziaService returns the singleton AmneziaService.
func GetAmneziaService() *AmneziaService { return amneziaServiceInstance }

// ------------------------------------------------------------------
// Key Generation
// ------------------------------------------------------------------

// GenerateKeypair generates a fresh WireGuard/AmneziaWG private+public keypair.
// Returns Base64-standard encoded strings (same format as `wg genkey` / `awg genkey`).
func (s *AmneziaService) GenerateKeypair() (privateKey, publicKey string, err error) {
	pk, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("amnezia: generate key: %w", err)
	}
	pub := pk.PublicKey()
	privateKey = base64.StdEncoding.EncodeToString(pk[:])
	publicKey = base64.StdEncoding.EncodeToString(pub[:])
	return
}

// DerivePublicKey derives the Curve25519 public key from a Base64-encoded private key.
func (s *AmneziaService) DerivePublicKey(privateKeyB64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return "", fmt.Errorf("amnezia: decode private key: %w", err)
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("amnezia: private key must be 32 bytes, got %d", len(raw))
	}
	// Curve25519 scalar multiplication to derive public key
	var priv, pub [32]byte
	copy(priv[:], raw)
	curve25519.ScalarBaseMult(&pub, &priv)
	return base64.StdEncoding.EncodeToString(pub[:]), nil
}

// ------------------------------------------------------------------
// Server Config
// ------------------------------------------------------------------

func (s *AmneziaService) GetConfig() (*AmneziaConfig, error) {
	val, err := s.ss.getString("amneziaConfig")
	if err != nil || val == "" {
		return nil, nil
	}
	cfg := &AmneziaConfig{}
	if err := json.Unmarshal([]byte(val), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *AmneziaService) SaveConfig(cfg *AmneziaConfig) error {
	// Auto-generate server keypair if needed
	if cfg.PrivateKey == "" || cfg.PrivateKey == "auto" {
		priv, pub, err := s.GenerateKeypair()
		if err != nil {
			return err
		}
		cfg.PrivateKey = priv
		cfg.PublicKey = pub
	} else if cfg.PublicKey == "" && cfg.PrivateKey != "" {
		pub, err := s.DerivePublicKey(cfg.PrivateKey)
		if err == nil {
			cfg.PublicKey = pub
		}
	}
	if cfg.Interface == "" {
		cfg.Interface = "awg0"
	}
	if cfg.MTU == 0 {
		cfg.MTU = 1420
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

func (s *AmneziaService) GetAllPeers() ([]model.AmneziaPeer, error) {
	db := database.GetDB()
	var peers []model.AmneziaPeer
	return peers, db.Order("id asc").Find(&peers).Error
}

func (s *AmneziaService) GetPeer(id uint) (*model.AmneziaPeer, error) {
	db := database.GetDB()
	var peer model.AmneziaPeer
	return &peer, db.First(&peer, id).Error
}

func (s *AmneziaService) AddPeer(peer *model.AmneziaPeer) error {
	// Auto-generate keypair
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
	// Auto-assign tunnel IP
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
	if s.IsRunning() {
		if err := s.liveAddPeer(peer); err != nil {
			logger.Warning("amnezia: live add peer: ", err)
		}
	}
	return nil
}

func (s *AmneziaService) EditPeer(peer *model.AmneziaPeer) error {
	db := database.GetDB()
	if err := db.Save(peer).Error; err != nil {
		return err
	}
	if s.IsRunning() {
		if err := s.liveAddPeer(peer); err != nil {
			logger.Warning("amnezia: live edit peer: ", err)
		}
	}
	return nil
}

func (s *AmneziaService) DeletePeer(id uint) error {
	peer, err := s.GetPeer(id)
	if err != nil {
		return err
	}
	if s.IsRunning() {
		if err := s.liveRemovePeer(peer.PublicKey); err != nil {
			logger.Warning("amnezia: live remove peer: ", err)
		}
	}
	db := database.GetDB()
	return db.Delete(&model.AmneziaPeer{}, id).Error
}

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
			_ = s.liveAddPeer(peer)
		} else {
			_ = s.liveRemovePeer(peer.PublicKey)
		}
	}
	return peer, nil
}

// ------------------------------------------------------------------
// Traffic & Expiry
// ------------------------------------------------------------------

// DepletePeers disables peers that have exceeded their data volume or expiry time.
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
			_ = s.liveRemovePeer(toDisable[i].PublicKey)
		}
		logger.Infof("AmneziaWG: disabled peer '%s' (depleted)", toDisable[i].Name)
	}
	return nil
}

// UpdateTrafficInDB adds the given rx/tx delta to a peer record by public key.
func (s *AmneziaService) UpdateTrafficInDB(pubKeyB64 string, rx, tx int64) {
	if rx == 0 && tx == 0 {
		return
	}
	database.GetDB().Model(&model.AmneziaPeer{}).
		Where("public_key = ? AND enable = true", pubKeyB64).
		Updates(map[string]interface{}{
			"down": gorm.Expr("down + ?", rx),
			"up":   gorm.Expr("up + ?", tx),
		})
}

// ------------------------------------------------------------------
// Client Config Generation
// ------------------------------------------------------------------

func (s *AmneziaService) GenerateClientConfig(peer *model.AmneziaPeer, serverIP string) (string, error) {
	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return "", fmt.Errorf("amnezia: server config not found")
	}
	serverPub := cfg.PublicKey
	if serverPub == "" {
		serverPub, err = s.DerivePublicKey(cfg.PrivateKey)
		if err != nil {
			return "", err
		}
	}

	mtu := cfg.MTU
	if mtu == 0 {
		mtu = 1420
	}
	clientAddr := strings.Replace(peer.AllowedIPs, "/32", "/24", 1)

	var sb strings.Builder
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", peer.PrivateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", clientAddr))
	sb.WriteString(fmt.Sprintf("MTU = %d\n", mtu))
	if cfg.DNS != "" {
		sb.WriteString(fmt.Sprintf("DNS = %s\n", cfg.DNS))
	}
	// AmneziaWG obfuscation params
	if cfg.Jc > 0 {
		sb.WriteString(fmt.Sprintf("Jc = %d\n", cfg.Jc))
		sb.WriteString(fmt.Sprintf("Jmin = %d\n", cfg.Jmin))
		sb.WriteString(fmt.Sprintf("Jmax = %d\n", cfg.Jmax))
		sb.WriteString(fmt.Sprintf("S1 = %d\n", cfg.S1))
		sb.WriteString(fmt.Sprintf("S2 = %d\n", cfg.S2))
		sb.WriteString(fmt.Sprintf("H1 = %d\n", cfg.H1))
		sb.WriteString(fmt.Sprintf("H2 = %d\n", cfg.H2))
		sb.WriteString(fmt.Sprintf("H3 = %d\n", cfg.H3))
		sb.WriteString(fmt.Sprintf("H4 = %d\n", cfg.H4))
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
	if cfg != nil && cfg.Address != "" {
		used[strings.Split(cfg.Address, "/")[0]] = true
	}
	for i := 2; i <= 254; i++ {
		candidate := fmt.Sprintf("%s.%d", base, i)
		if !used[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available IP in %s.0/24", base)
}
