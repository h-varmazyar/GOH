package proxy

import "github.com/h-varmazyar/gatec/internal/models"

// --- Request DTOs ---

type createProxyReq struct {
	Server   string           `json:"server" binding:"required" example:"tanzibjf.club"`
	Port     int32            `json:"port" binding:"required,min=1,max=65535" example:"443"`
	Type     models.ProxyType `json:"type" binding:"required" example:"mtproto"`
	Secret   string           `json:"secret,omitempty" example:"ee1603010200010001fc030386e24c3add626973636f7474692e79656b74616e65742e636f6d"`
	Username string           `json:"username,omitempty" example:"proxyuser"`
	Password string           `json:"password,omitempty" example:"proxypass"`
	HTTPOnly bool             `json:"http_only,omitempty" example:"false"`
	IsEnable bool             `json:"is_enable" example:"true"`
}

func (r *createProxyReq) ToModel() *models.Proxy {
	return &models.Proxy{
		Server:   r.Server,
		Port:     r.Port,
		Type:     r.Type,
		Secret:   r.Secret,
		Username: r.Username,
		Password: r.Password,
		HTTPOnly: r.HTTPOnly,
		IsEnable: r.IsEnable,
	}
}

type updateProxyReq struct {
	Server   string           `json:"server" binding:"required" example:"tanzibjf.club"`
	Port     int32            `json:"port" binding:"required,min=1,max=65535" example:"443"`
	Type     models.ProxyType `json:"type" binding:"required" example:"mtproto"`
	Secret   string           `json:"secret,omitempty" example:"ee1603010200010001fc030386e24c3add626973636f7474692e79656b74616e65742e636f6d"`
	Username string           `json:"username,omitempty" example:"proxyuser"`
	Password string           `json:"password,omitempty" example:"proxypass"`
	HTTPOnly bool             `json:"http_only,omitempty" example:"false"`
	IsEnable bool             `json:"is_enable" example:"true"`
}

func (r *updateProxyReq) ToModel() *models.Proxy {
	return &models.Proxy{
		Server:   r.Server,
		Port:     r.Port,
		Type:     r.Type,
		Secret:   r.Secret,
		Username: r.Username,
		Password: r.Password,
		HTTPOnly: r.HTTPOnly,
		IsEnable: r.IsEnable,
	}
}

type toggleProxyReq struct {
	Enable bool `json:"enable" binding:"required" example:"true"`
}

// --- Response DTOs ---

type proxyStatsResp struct {
	Total       int `json:"total" example:"10"`
	Enabled     int `json:"enabled" example:"8"`
	Active      int `json:"active" example:"6"`
	Assignments int `json:"assignments" example:"15"`
}

type errorResp struct {
	Error string `json:"error" example:"server is required"`
}
