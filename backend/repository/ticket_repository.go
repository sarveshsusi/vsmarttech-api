package repository

import (
	"fmt"
	"rbac/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

/* ======================
   CREATE
====================== */

func (r *TicketRepository) Create(ticket *models.Ticket) error {
	return r.db.Create(ticket).Error
}

/* ======================
   GENERATE TICKET ID
/* ======================
   GENERATE TICKET ID
====================== */

// GenerateNextTicketID generates ticket ID in format VS/MM/YY/number
// Example: VS/04/26/10 for 10th ticket in April 2026
func (r *TicketRepository) GenerateNextTicketID() (string, error) {
	now := time.Now()
	month := fmt.Sprintf("%02d", now.Month())
	year := fmt.Sprintf("%02d", now.Year()%100)
	
	// Get count of tickets for current month/year
	prefix := fmt.Sprintf("VS/%s/%s/", month, year)
	
	var count int64
	err := r.db.Model(&models.Ticket{}).
		Where("id LIKE ?", prefix+"%").
		Count(&count).Error
	
	if err != nil {
		return "", err
	}
	
	// Next ticket number for this month
	nextNumber := count + 1
	
	ticketID := fmt.Sprintf("VS/%s/%s/%d", month, year, nextNumber)
	return ticketID, nil
}

/* ======================
   GETTERS
====================== */

func (r *TicketRepository) GetByID(id string) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.
		Preload("Attachments").
		Preload("SupportEngineer").
		Preload("SupportEngineer.User").
		First(&ticket, "id = ?", id).Error
	return &ticket, err
}

func (r *TicketRepository) GetAll() ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.
		Preload("Customer", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Company")
		}).
		Preload("CustomerSolution").
		Preload("CustomerSolution.Solution").
		Preload("Attachments").
		Preload("SupportEngineer").
		Preload("SupportEngineer.User").
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

func (r *TicketRepository) GetByCustomerID(customerID uuid.UUID) ([]models.Ticket, error) {
	var tickets []models.Ticket

	err := r.db.
		Preload("Customer").
		Preload("Attachments").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&tickets).Error

	return tickets, err
}

func (r *TicketRepository) GetByEngineerID(engineerID uuid.UUID) ([]models.Ticket, error) {
	var tickets []models.Ticket
	err := r.db.
		Preload("SupportEngineer").
		Preload("SupportEngineer.User").
		Preload("Customer").
		Preload("CustomerSolution").
		Preload("CustomerSolution.Solution").
		Preload("Attachments").
		Where("engineer_id = ?", engineerID).
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

/* ======================
   ASSIGNMENT HELPERS
====================== */

func (r *TicketRepository) IsAssigned(ticketID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.TicketAssignment{}).
		Where("ticket_id = ?", ticketID).
		Count(&count).Error
	return count > 0, err
}

func (r *TicketRepository) SupportEngineerExists(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.SupportEngineer{}).
		Where("id = ? AND is_active = true", id).
		Count(&count).Error
	return count > 0, err
}

/* ======================
   MUTATIONS
====================== */

func (r *TicketRepository) UpdateFields(
	ticketID string,
	fields map[string]interface{},
) error {
	return r.db.Model(&models.Ticket{}).
		Where("id = ?", ticketID).
		Updates(fields).Error
}

func (r *TicketRepository) AssignEngineer(
	assignment *models.TicketAssignment,
) error {
	return r.db.Create(assignment).Error
}

func (r *TicketRepository) CreateStatusHistory(
	history *models.TicketStatusHistory,
) error {
	return r.db.Create(history).Error
}

func (r *TicketRepository) FindOverdueTickets(
	days int,
) ([]models.Ticket, error) {

	var tickets []models.Ticket

	cutoff := time.Now().AddDate(0, 0, -days)

	err := r.db.
		Where("status IN ?", []models.TicketStatus{
			models.StatusOpen,
			models.StatusAssigned,
			models.StatusInProgress,
		}).
		Where("created_at <= ?", cutoff).
		Find(&tickets).
		Error

	return tickets, err
}

func (r *TicketRepository) CreateTx(
	tx *gorm.DB,
	ticket *models.Ticket,
) error {
	return tx.Create(ticket).Error
}

func (r *TicketRepository) AssignEngineerTx(
	tx *gorm.DB,
	ticketID string,
	engineerID uuid.UUID,
	adminID uuid.UUID,
) error {

	now := time.Now()

	// 1️⃣ Create assignment row
	if err := tx.Create(&models.TicketAssignment{
		TicketID:   ticketID,
		EngineerID: engineerID,
		AssignedBy: adminID,
		AssignedAt: now,
	}).Error; err != nil {
		return err
	}

	// 2️⃣ Update ticket status
	return tx.Model(&models.Ticket{}).
		Where("id = ?", ticketID).
		Updates(map[string]interface{}{
			"status":     models.StatusAssigned,
			"updated_at": now,
		}).Error
}
