package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Entity      string    `json:"entity"`
	EntityID    uuid.UUID `gorm:"type:uuid" json:"entity_id"`
	Action      string    `json:"action"`
	PerformedBy uuid.UUID `gorm:"type:uuid" json:"performed_by"`
	IP          string    `json:"ip"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
