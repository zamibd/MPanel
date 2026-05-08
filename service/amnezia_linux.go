//go:build linux
// +build linux

package service

// amnezia_linux.go – Linux interface management using amneziawg-go directly.
// Uses the UAPI (userspace WireGuard API) protocol via device.IpcSet/IpcGet.

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"

	awgConn "github.com/amnezia-vpn/amneziawg-go/conn"
	awgDevice "github.com/amnezia-vpn/amneziawg-go/device"
	awgTun "github.com/amnezia-vpn/amneziawg-go/tun"

	"github.com/zamibd/MPanel/database/model"
	"github.com/zamibd/MPanel/logger"
)

// StartInterface creates and brings up the AmneziaWG userspace TUN device.
func (s *AmneziaService) StartInterface() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.GetConfig()
	if err != nil || cfg == nil {
		return fmt.Errorf("amnezia: server config not found – POST /amnezia/config first")
	}
	s.cfg = cfg

	iface := cfg.Interface
	if iface == "" {
		iface = "awg0"
		cfg.Interface = iface
	}
	mtu := cfg.MTU
	if mtu == 0 {
		mtu = 1420
	}

	// Create userspace TUN device
	tunDev, err := awgTun.CreateTUN(iface, mtu)
	if err != nil {
		return fmt.Errorf("amnezia: create TUN %s: %w", iface, err)
	}

	// Create AmneziaWG device (userspace WireGuard engine)
	awgLog := awgDevice.NewLogger(awgDevice.LogLevelError, fmt.Sprintf("[%s] ", iface))
	dev := awgDevice.NewDevice(tunDev, awgConn.NewStdNetBind(), awgLog)

	// Configure private key + listen port via UAPI
	uapiCfg, err := s.buildServerUAPI(cfg)
	if err != nil {
		dev.Close()
		return fmt.Errorf("amnezia: build uapi config: %w", err)
	}
	if err := dev.IpcSet(uapiCfg); err != nil {
		dev.Close()
		return fmt.Errorf("amnezia: ipc set config: %w", err)
	}

	// Bring device up
	if err := dev.Up(); err != nil {
		dev.Close()
		return fmt.Errorf("amnezia: device up: %w", err)
	}

	// Assign IP to interface
	if err := runIPCmd("addr", "add", cfg.Address, "dev", iface); err != nil {
		logger.Warning("amnezia: assign IP: ", err)
	}
	if err := runIPCmd("link", "set", "up", "dev", iface); err != nil {
		logger.Warning("amnezia: link up: ", err)
	}
	if cfg.PostUp != "" {
		runShell(cfg.PostUp)
	}

	s.deviceHandle = dev
	s.running = true

	// Apply all enabled peers
	peers, _ := s.GetAllPeers()
	for i := range peers {
		if peers[i].Enable {
			_ = s.liveAddPeer(&peers[i])
		}
	}

	logger.Infof("AmneziaWG interface %s started (userspace amneziawg-go)", iface)
	return nil
}

// StopInterface shuts down the AmneziaWG device and removes the interface.
func (s *AmneziaService) StopInterface() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg != nil && s.cfg.PostDown != "" {
		runShell(s.cfg.PostDown)
	}
	if s.deviceHandle != nil {
		s.deviceHandle.(*awgDevice.Device).Close()
		s.deviceHandle = nil
	}
	iface := "awg0"
	if s.cfg != nil && s.cfg.Interface != "" {
		iface = s.cfg.Interface
	}
	_ = runIPCmd("link", "del", iface)
	s.running = false
	logger.Info("AmneziaWG interface stopped")
	return nil
}

func (s *AmneziaService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// SyncTraffic reads per-peer traffic from the UAPI and persists to DB.
func (s *AmneziaService) SyncTraffic() error {
	s.mu.RLock()
	dh := s.deviceHandle
	s.mu.RUnlock()
	if dh == nil {
		return nil
	}
	dev := dh.(*awgDevice.Device)

	uapiOutput, err := dev.IpcGet()
	if err != nil {
		return fmt.Errorf("amnezia: ipc get: %w", err)
	}

	// UAPI output format: key=value lines, peers separated by blank lines.
	// Relevant per-peer fields: public_key (hex), rx_bytes, tx_bytes
	type peerStat struct {
		pubKeyHex string
		rx, tx    int64
	}
	var stats []peerStat
	var cur peerStat

	for _, line := range strings.Split(uapiOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if cur.pubKeyHex != "" {
				stats = append(stats, cur)
				cur = peerStat{}
			}
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := parts[0], parts[1]
		switch k {
		case "public_key":
			cur.pubKeyHex = v
		case "rx_bytes":
			fmt.Sscanf(v, "%d", &cur.rx)
		case "tx_bytes":
			fmt.Sscanf(v, "%d", &cur.tx)
		}
	}
	if cur.pubKeyHex != "" {
		stats = append(stats, cur)
	}

	for _, st := range stats {
		// UAPI uses lowercase hex; convert to Base64 for DB lookup
		raw, err := hex.DecodeString(st.pubKeyHex)
		if err != nil {
			continue
		}
		pubB64 := base64.StdEncoding.EncodeToString(raw)
		s.UpdateTrafficInDB(pubB64, st.rx, st.tx)
	}
	return nil
}

// liveAddPeer applies a single peer to the running device via UAPI.
func (s *AmneziaService) liveAddPeer(peer *model.AmneziaPeer) error {
	s.mu.RLock()
	dh := s.deviceHandle
	s.mu.RUnlock()
	if dh == nil {
		return nil
	}
	dev := dh.(*awgDevice.Device)

	pubRaw, err := base64.StdEncoding.DecodeString(peer.PublicKey)
	if err != nil {
		return fmt.Errorf("amnezia: decode peer pubkey: %w", err)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("public_key=%s\n", hex.EncodeToString(pubRaw)))
	sb.WriteString("replace_allowed_ips=true\n")
	for _, cidr := range strings.Split(peer.AllowedIPs, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr != "" {
			sb.WriteString(fmt.Sprintf("allowed_ip=%s\n", cidr))
		}
	}
	return dev.IpcSet(sb.String())
}

// liveRemovePeer removes a peer from the running device via UAPI.
func (s *AmneziaService) liveRemovePeer(pubKeyB64 string) error {
	s.mu.RLock()
	dh := s.deviceHandle
	s.mu.RUnlock()
	if dh == nil {
		return nil
	}
	dev := dh.(*awgDevice.Device)

	pubRaw, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return fmt.Errorf("amnezia: decode peer pubkey for removal: %w", err)
	}
	return dev.IpcSet(fmt.Sprintf("public_key=%s\nremove=true\n", hex.EncodeToString(pubRaw)))
}

// buildServerUAPI builds the UAPI configuration string for the server device.
func (s *AmneziaService) buildServerUAPI(cfg *AmneziaConfig) (string, error) {
	privRaw, err := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("amnezia: decode server private key: %w", err)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("private_key=%s\n", hex.EncodeToString(privRaw)))
	sb.WriteString(fmt.Sprintf("listen_port=%d\n", cfg.ListenPort))

	// AmneziaWG-specific obfuscation params (passed as UAPI extensions)
	if cfg.Jc > 0 {
		sb.WriteString(fmt.Sprintf("jc=%d\n", cfg.Jc))
		sb.WriteString(fmt.Sprintf("jmin=%d\n", cfg.Jmin))
		sb.WriteString(fmt.Sprintf("jmax=%d\n", cfg.Jmax))
		sb.WriteString(fmt.Sprintf("s1=%d\n", cfg.S1))
		sb.WriteString(fmt.Sprintf("s2=%d\n", cfg.S2))
		sb.WriteString(fmt.Sprintf("h1=%d\n", cfg.H1))
		sb.WriteString(fmt.Sprintf("h2=%d\n", cfg.H2))
		sb.WriteString(fmt.Sprintf("h3=%d\n", cfg.H3))
		sb.WriteString(fmt.Sprintf("h4=%d\n", cfg.H4))
	}
	return sb.String(), nil
}

func runIPCmd(args ...string) error {
	out, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip %s: %s (%w)", strings.Join(args, " "), out, err)
	}
	return nil
}

func runShell(cmd string) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		logger.Warning("amnezia shell: ", string(out), " – ", err)
	}
}

