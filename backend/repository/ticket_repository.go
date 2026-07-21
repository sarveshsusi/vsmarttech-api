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

/* ======================
   TICKET EVENTS
====================== */

func (r *TicketRepository) CreateEvent(event *models.TicketEvent) error {
	return r.db.Create(event).Error
}

func (r *TicketRepository) CreateEventTx(tx *gorm.DB, event *models.TicketEvent) error {
	return tx.Create(event).Error
}

func (r *TicketRepository) ListEventsByTicketID(ticketID string) ([]models.TicketEvent, error) {
	var events []models.TicketEvent
	err := r.db.
		Preload("Actor").
		Preload("FromEngineer").
		Preload("FromEngineer.User").
		Preload("ToEngineer").
		Preload("ToEngineer.User").
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Find(&events).Error
	return events, err
}

type TicketStatusListFilter struct {
	Status     string
	CompanyID  *uuid.UUID
	EngineerID *uuid.UUID
	StartDate  *time.Time
	EndDate    *time.Time
	Search     string
}

func (r *TicketRepository) ListForStatusPage(filter TicketStatusListFilter) ([]models.Ticket, error) {
	q := r.db.Model(&models.Ticket{}).
		Preload("Customer", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Company")
		}).
		Preload("SupportEngineer").
		Preload("SupportEngineer.User")

	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.EngineerID != nil {
		q = q.Where("engineer_id = ?", *filter.EngineerID)
	}
	if filter.StartDate != nil {
		q = q.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		q = q.Where("created_at <= ?", *filter.EndDate)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("(id ILIKE ? OR title ILIKE ?)", like, like)
	}
	if filter.CompanyID != nil {
		q = q.Joins("JOIN customers ON customers.id = tickets.customer_id").
			Where("customers.company_id = ?", *filter.CompanyID)
	}

	var tickets []models.Ticket
	err := q.Order("updated_at DESC").Find(&tickets).Error
	return tickets, err
}

func (r *TicketRepository) GetLatestEventsByTicketIDs(ticketIDs []string) (map[string]models.TicketEvent, error) {
	result := make(map[string]models.TicketEvent)
	if len(ticketIDs) == 0 {
		return result, nil
	}

	var events []models.TicketEvent
	err := r.db.
		Preload("Actor").
		Preload("ToEngineer").
		Preload("ToEngineer.User").
		Where("ticket_id IN ?", ticketIDs).
		Order("created_at DESC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}

	for _, ev := range events {
		if _, exists := result[ev.TicketID]; !exists {
			result[ev.TicketID] = ev
		}
	}
	return result, nil
}

func (r *TicketRepository) CountReopensByTicketIDs(ticketIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(ticketIDs) == 0 {
		return result, nil
	}

	type row struct {
		TicketID string
		Count    int64
	}
	var rows []row
	err := r.db.Model(&models.TicketEvent{}).
		Select("ticket_id, COUNT(*) as count").
		Where("ticket_id IN ? AND event_type = ?", ticketIDs, models.TicketEventReopened).
		Group("ticket_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.TicketID] = r.Count
	}
	return result, nil
}
