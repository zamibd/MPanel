package api

import (
	"strconv"

	"github.com/zamibd/MPanel/database/model"
	"github.com/zamibd/MPanel/service"
	"github.com/zamibd/MPanel/util/common"

	"github.com/gin-gonic/gin"
)

// AmneziaHandler handles all /api/amnezia/* and /apiv2/amnezia/* routes.
// It is mounted independently from the main ApiService so it can be shared
// between the cookie-session (v1) and token-authenticated (v2) APIs.
type AmneziaHandler struct {
	svc *service.AmneziaService
}

func NewAmneziaHandler(v1 *gin.RouterGroup, v2 *gin.RouterGroup) {
	h := &AmneziaHandler{svc: service.GetAmneziaService()}
	h.initRoutes(v1)
	h.initRoutes(v2)
}

func (h *AmneziaHandler) initRoutes(g *gin.RouterGroup) {
	ag := g.Group("/amnezia")
	// Config
	ag.GET("/config", h.getConfig)
	ag.POST("/config", h.saveConfig)
	// Interface lifecycle
	ag.POST("/start", h.startInterface)
	ag.POST("/stop", h.stopInterface)
	ag.GET("/status", h.getStatus)
	// Peer management
	ag.GET("/peers", h.listPeers)
	ag.GET("/peers/:id", h.getPeer)
	ag.POST("/peers", h.addPeer)
	ag.PUT("/peers/:id", h.editPeer)
	ag.DELETE("/peers/:id", h.deletePeer)
	ag.POST("/peers/:id/toggle", h.togglePeer)
	// Client config download
	ag.GET("/peers/:id/config", h.peerConfig)
	// Key generation
	ag.GET("/keypair", h.genKeypair)
}

// GET /amnezia/config
func (h *AmneziaHandler) getConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	// Redact private key from response
	if cfg != nil {
		cfg.PrivateKey = "***"
	}
	jsonObj(c, cfg, nil)
}

// POST /amnezia/config
func (h *AmneziaHandler) saveConfig(c *gin.Context) {
	var cfg service.AmneziaConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		jsonMsg(c, "", err)
		return
	}
	// Auto-generate server keypair if private key is "auto" or empty
	if cfg.PrivateKey == "" || cfg.PrivateKey == "auto" {
		priv, pub, err := h.svc.GenerateKeypair()
		if err != nil {
			jsonMsg(c, "", err)
			return
		}
		cfg.PrivateKey = priv
		cfg.PublicKey = pub
	}
	if err := h.svc.SaveConfig(&cfg); err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonMsg(c, "amnezia config saved", nil)
}

// POST /amnezia/start
func (h *AmneziaHandler) startInterface(c *gin.Context) {
	if err := h.svc.StartInterface(); err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonMsg(c, "AmneziaWG interface started", nil)
}

// POST /amnezia/stop
func (h *AmneziaHandler) stopInterface(c *gin.Context) {
	if err := h.svc.StopInterface(); err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonMsg(c, "AmneziaWG interface stopped", nil)
}

// GET /amnezia/status
func (h *AmneziaHandler) getStatus(c *gin.Context) {
	jsonObj(c, map[string]interface{}{
		"running": h.svc.IsRunning(),
	}, nil)
}

// GET /amnezia/peers
func (h *AmneziaHandler) listPeers(c *gin.Context) {
	peers, err := h.svc.GetAllPeers()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, peers, nil)
}

// GET /amnezia/peers/:id
func (h *AmneziaHandler) getPeer(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	peer, err := h.svc.GetPeer(id)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, peer, nil)
}

// POST /amnezia/peers
// Body: AmneziaPeer JSON. Set privateKey to "auto" for auto key generation.
func (h *AmneziaHandler) addPeer(c *gin.Context) {
	var peer model.AmneziaPeer
	if err := c.ShouldBindJSON(&peer); err != nil {
		jsonMsg(c, "", err)
		return
	}
	peer.Id = 0 // ensure new record
	if err := h.svc.AddPeer(&peer); err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, peer, nil)
}

// PUT /amnezia/peers/:id
func (h *AmneziaHandler) editPeer(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	var peer model.AmneziaPeer
	if err := c.ShouldBindJSON(&peer); err != nil {
		jsonMsg(c, "", err)
		return
	}
	peer.Id = id
	if err := h.svc.EditPeer(&peer); err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, peer, nil)
}

// DELETE /amnezia/peers/:id
func (h *AmneziaHandler) deletePeer(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	if err := h.svc.DeletePeer(id); err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonMsg(c, "peer deleted", nil)
}

// POST /amnezia/peers/:id/toggle
func (h *AmneziaHandler) togglePeer(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	peer, err := h.svc.TogglePeer(id)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, peer, nil)
}

// GET /amnezia/peers/:id/config?server=<ip>
// Returns a ready-to-use WireGuard .conf file for the client.
func (h *AmneziaHandler) peerConfig(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	peer, err := h.svc.GetPeer(id)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	serverIP := c.Query("server")
	if serverIP == "" {
		serverIP = getHostname(c)
	}
	conf, err := h.svc.GenerateClientConfig(peer, serverIP)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+peer.Name+".conf")
	c.String(200, conf)
}

// GET /amnezia/keypair
// Returns a freshly generated private/public keypair.
func (h *AmneziaHandler) genKeypair(c *gin.Context) {
	priv, pub, err := h.svc.GenerateKeypair()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, map[string]string{
		"privateKey": priv,
		"publicKey":  pub,
	}, nil)
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

func parseID(c *gin.Context) (uint, error) {
	raw := c.Param("id")
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, common.NewError("invalid id: ", raw)
	}
	return uint(n), nil
}

