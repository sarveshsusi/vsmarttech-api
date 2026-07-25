package repository

import (
	"rbac/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PushSubscriptionRepository struct {
	db *gorm.DB
}

func NewPushSubscriptionRepository(db *gorm.DB) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{db: db}
}

// Upsert creates or updates a subscription keyed by endpoint.
func (r *PushSubscriptionRepository) Upsert(sub *models.PushSubscription) error {
	now := time.Now()
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	sub.UpdatedAt = now
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "endpoint"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "p256dh", "auth", "user_agent", "updated_at",
		}),
	}).Create(sub).Error
}

func (r *PushSubscriptionRepository) ListByUserID(userID uuid.UUID) ([]models.PushSubscription, error) {
	var subs []models.PushSubscription
	err := r.db.Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}

func (r *PushSubscriptionRepository) DeleteByEndpoint(userID uuid.UUID, endpoint string) error {
	return r.db.
		Where("user_id = ? AND endpoint = ?", userID, endpoint).
		Delete(&models.PushSubscription{}).Error
}

func (r *PushSubscriptionRepository) DeleteByID(id uuid.UUID) error {
	return r.db.Delete(&models.PushSubscription{}, "id = ?", id).Error
}

func (r *PushSubscriptionRepository) DeleteByEndpointOnly(endpoint string) error {
	return r.db.Where("endpoint = ?", endpoint).Delete(&models.PushSubscription{}).Error
}
