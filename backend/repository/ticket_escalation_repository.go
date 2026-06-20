package repository

import (
	"rbac/models"
	"gorm.io/gorm"
)

type TicketEscalationRepository struct {
	db *gorm.DB
}

func NewTicketEscalationRepository(db *gorm.DB) *TicketEscalationRepository {
	return &TicketEscalationRepository{db: db}
}

func (r *TicketEscalationRepository) AlreadyEscalated(
	ticketID string,
) (bool, error) {

	var count int64
	err := r.db.
		Model(&models.TicketEscalation{}).
		Where("ticket_id = ? AND resolved = false", ticketID).
		Count(&count).Error

	return count > 0, err
}

func (r *TicketEscalationRepository) Create(
	ticketID string,
) error {

	return r.db.Create(&models.TicketEscalation{
		TicketID: ticketID,
		Resolved: false,
	}).Error
}

func (r *TicketEscalationRepository) ResolveByTicket(
	ticketID string,
) error {

	return r.db.
		Model(&models.TicketEscalation{}).
		Where("ticket_id = ? AND resolved = false", ticketID).
		Update("resolved", true).
		Error
}
