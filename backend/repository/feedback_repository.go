package repository

import (
	"rbac/models"

	"gorm.io/gorm"
)

type FeedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Create(
	feedback *models.TicketFeedback,
) error {
	return r.db.Create(feedback).Error
}
