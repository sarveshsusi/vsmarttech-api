package repository

import (
	"time"

	"rbac/models"

	"gorm.io/gorm"
)

type EscalationRepository struct {
	db *gorm.DB
}

func NewEscalationRepository(db *gorm.DB) *EscalationRepository {
	return &EscalationRepository{db: db}
}

/*
	=====================
	  Find overdue tickets

=====================
*/
func (r *EscalationRepository) FindOverdueTickets(
	olderThan time.Time,
) ([]models.Ticket, error) {

	var tickets []models.Ticket

	err := r.db.
		Where(`
			status != ? 
			AND created_at <= ?
			AND id NOT IN (
				SELECT ticket_id 
				FROM ticket_escalations 
				WHERE resolved = false
			)
		`,
			models.StatusClosed,
			olderThan,
		).
		Find(&tickets).Error

	return tickets, err
}

/*
	=====================
	  Mark escalated

=====================
*/
func (r *EscalationRepository) MarkEscalated(
	ticketID string,
) error {

	return r.db.Create(&models.TicketEscalation{
		TicketID:    ticketID,
		EscalatedAt: time.Now(),
		Resolved:    false,
	}).Error
}
