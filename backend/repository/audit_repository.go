package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Log(
	entity string,
	entityID uuid.UUID,
	action string,
	performedBy uuid.UUID,
	ip string,
	userAgent string,
) error {

	return r.db.Create(&models.AuditLog{
		Entity:      entity,
		EntityID:   entityID,
		Action:     action,
		PerformedBy: performedBy,
		IP:         ip,
		UserAgent:  userAgent,
	}).Error
}
