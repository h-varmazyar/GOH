package models

import (
	"github.com/h-varmazyar/gatec/internal/consts"
	"gorm.io/gorm"
)

type Asset struct {
	gorm.Model
	Name  string
	Label string
	Type  consts.AssetType
}
