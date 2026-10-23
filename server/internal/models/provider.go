package models

import (
	"github.com/h-varmazyar/gatec/internal/consts"
	"gorm.io/gorm"
)

type Provider struct {
	gorm.Model
	Name   consts.Provider
	Status consts.ProviderStatus
	APIKey string
}
