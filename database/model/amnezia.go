package model

// AmneziaPeer represents a single WireGuard/AmneziaWG peer managed by MPanel.
// Each peer has its own keypair, IP address, expiry, and traffic accounting.
type AmneziaPeer struct {
	Id         uint   `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Enable     bool   `json:"enable" form:"enable" gorm:"default:true;not null"`
	Name       string `json:"name" form:"name" gorm:"not null"`
	Desc       string `json:"desc" form:"desc"`
	Group      string `json:"group" form:"group"`
	PublicKey  string `json:"publicKey" form:"publicKey" gorm:"unique;not null"`
	PrivateKey string `json:"privateKey" form:"privateKey"`
	// AllowedIPs is the tunnel IP assigned to this peer, e.g. "10.8.0.2/32"
	AllowedIPs string `json:"allowedIPs" form:"allowedIPs" gorm:"not null"`

	// Traffic accounting (bytes)
	Up        int64 `json:"up" form:"up" gorm:"default:0;not null"`
	Down      int64 `json:"down" form:"down" gorm:"default:0;not null"`
	TotalUp   int64 `json:"totalUp" form:"totalUp" gorm:"default:0;not null"`
	TotalDown int64 `json:"totalDown" form:"totalDown" gorm:"default:0;not null"`

	// Limits
	Volume int64 `json:"volume" form:"volume" gorm:"default:0;not null"` // 0 = unlimited (bytes)
	Expiry int64 `json:"expiry" form:"expiry" gorm:"default:0;not null"` // 0 = never (unix timestamp)

	// Auto-reset fields (similar to Client)
	AutoReset bool  `json:"autoReset" form:"autoReset" gorm:"default:false;not null"`
	ResetDays int   `json:"resetDays" form:"resetDays" gorm:"default:0;not null"`
	NextReset int64 `json:"nextReset" form:"nextReset" gorm:"default:0;not null"`

	CreatedAt int64 `json:"createdAt" gorm:"autoCreateTime"`
}
