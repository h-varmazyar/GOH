package models

import (
	"github.com/h-varmazyar/gatec/internal/consts"
	"gorm.io/gorm"
)

type Strategy struct {
	gorm.Model
	UserID      uint
	Name        string
	Description string
	Provider    *Provider
	Status      consts.StrategyStatus
	Markets     []*Market
}
