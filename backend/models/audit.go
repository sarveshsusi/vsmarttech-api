package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Entity      string
	EntityID    uuid.UUID `gorm:"type:uuid"`
	Action      string
	PerformedBy uuid.UUID `gorm:"type:uuid"`
	IP          string
	UserAgent   string
	CreatedAt   time.Time
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
