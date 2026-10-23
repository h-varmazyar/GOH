package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Mobile         string `gorm:"uniqueIndex;not null"`
	Username       string `gorm:"uniqueIndex;not null"`
	HashedPassword string `gorm:"not null"`
	FirstName      string
	LastName       string
}
