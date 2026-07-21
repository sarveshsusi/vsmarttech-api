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

func (r *AuditRepository) DB() *gorm.DB {
	return r.db
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

type AuditListFilter struct {
	Search    string
	UserID    string
	StartDate string
	EndDate   string
	Limit     int
	Offset    int
}

func (r *AuditRepository) List(filter AuditListFilter) ([]models.AuditLog, int64, error) {
	q := r.db.Model(&models.AuditLog{})

	// Hide read-only HTTP noise from older middleware that logged every GET.
	q = q.Where(
		"action NOT ILIKE 'GET %' AND action NOT ILIKE 'HEAD %' AND action NOT ILIKE 'OPTIONS %'",
	)

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("action ILIKE ? OR ip ILIKE ?", like, like)
	}
	if filter.UserID != "" {
		if uid, err := uuid.Parse(filter.UserID); err == nil {
			q = q.Where("performed_by = ?", uid)
		}
	}
	if filter.StartDate != "" {
		q = q.Where("created_at >= ?", filter.StartDate+" 00:00:00")
	}
	if filter.EndDate != "" {
		q = q.Where("created_at <= ?", filter.EndDate+" 23:59:59")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var rows []models.AuditLog
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}
