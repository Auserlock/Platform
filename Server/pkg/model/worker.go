package model

import (
	"time"

	"gorm.io/gorm"
)

type Worker struct {
	gorm.Model

	WorkerID string `gorm:"uniqueIndex;not null"`
	APIKey   string `gorm:"not null"`
	Hostname string
	Status   string `gorm:"default:'offline';not null"`
	LastSeen *time.Time
}
