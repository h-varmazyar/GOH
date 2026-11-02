package models

import (
	"time"

	"github.com/h-varmazyar/gatec/internal/consts"
)

// TSEMarketIssue is a problem of the TSE catalog shown in the panel, so the user
// can decide on it.
type TSEMarketIssue struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	// Key identifies the issue across detections, so that a decision on it is kept.
	Key      string                  `gorm:"uniqueIndex;not null"`
	Type     consts.TSEIssueType     `gorm:"index;not null"`
	Severity consts.TSEIssueSeverity `gorm:"index;not null"`
	Status   consts.TSEIssueStatus   `gorm:"index;not null;default:OPEN"`
	Symbol   string                  `gorm:"index"`
	Name     string
	Detail   string
	// InsCodes are the insCodes the issue is about.
	InsCodes []string `gorm:"type:jsonb;serializer:json"`
	// MarketIDs are the markets the issue is about; their insCodes are shown with it.
	MarketIDs  []uint `gorm:"type:jsonb;serializer:json"`
	Note       string
	DetectedAt time.Time `gorm:"index"`
	ResolvedAt *time.Time
}
