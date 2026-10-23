package models

import (
	"time"

	"gorm.io/gorm"
)

// ProxyType represents the type of proxy
type ProxyType string

const (
	ProxyTypeMTProto ProxyType = "mtproto"
	ProxyTypeSocks5  ProxyType = "socks5"
	ProxyTypeHTTP    ProxyType = "http"
)

// ProxyStatus represents the current health status of a proxy
type ProxyStatus string

const (
	ProxyStatusActive   ProxyStatus = "active"
	ProxyStatusInactive ProxyStatus = "inactive"
	ProxyStatusUnknown  ProxyStatus = "unknown"
)

// Proxy represents a proxy server used for Telegram connections
type Proxy struct {
	gorm.Model
	Server   string      `gorm:"not null" json:"server"`
	Port     int32       `gorm:"not null" json:"port"`
	Type     ProxyType   `gorm:"not null;default:'mtproto'" json:"type"`
	Status   ProxyStatus `gorm:"not null;default:'unknown'" json:"status"`
	IsEnable bool        `gorm:"not null;default:true" json:"is_enable"`

	// MTProto specific
	Secret string `gorm:"" json:"secret,omitempty"`

	// SOCKS5/HTTP specific
	Username string `gorm:"" json:"username,omitempty"`
	Password string `gorm:"" json:"password,omitempty"`

	// HTTP specific
	HTTPOnly bool `gorm:"not null;default:false" json:"http_only,omitempty"`

	// Health tracking
	LastCheckedAt    *time.Time `gorm:"" json:"last_checked_at"`
	LastSuccessAt    *time.Time `gorm:"" json:"last_success_at"`
	FailCount        int        `gorm:"not null;default:0" json:"fail_count"`
	ConsecutiveFails int        `gorm:"not null;default:0" json:"consecutive_fails"`

	// Usage tracking
	AssignedAccounts int `gorm:"not null;default:0" json:"assigned_accounts"`
}

// ProxyAssignment tracks which proxy is assigned to which telegram account
type ProxyAssignment struct {
	gorm.Model
	AccountID string `gorm:"not null;index" json:"account_id"`
	ProxyID   uint   `gorm:"not null;index" json:"proxy_id"`
	Proxy     Proxy  `gorm:"foreignKey:ProxyID" json:"proxy"`
	IsActive  bool   `gorm:"not null;default:true" json:"is_active"`
}
