package models

import (
	"time"

	"github.com/google/uuid"
)

type RememberedDevice struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	Token     string    `gorm:"not null;uniqueIndex"`
	UserAgent string
	IPAddress string
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (RememberedDevice) TableName() string {
	return "remembered_devices"
}
