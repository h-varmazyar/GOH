package models

import "gorm.io/gorm"

const (
	DynamicConfigSectionTabdeal = "adapter.tabdeal"
	DynamicConfigKeyAuthToken   = "auth_token"
)

// DynamicConfig stores a runtime configuration value under a named section.
type DynamicConfig struct {
	gorm.Model
	Section string `gorm:"not null;default:'';uniqueIndex:uidx_dynamic_configs_section_key" json:"section"`
	Key     string `gorm:"not null;uniqueIndex:uidx_dynamic_configs_section_key" json:"key"`
	Value   string `gorm:"type:text;not null" json:"-"`
}
