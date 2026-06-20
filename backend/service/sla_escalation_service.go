package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"rbac/models"
	"rbac/repository"
	"rbac/utils"

	"gorm.io/gorm"
)

// SLAEscalationService handles SLA escalation logic
type SLAEscalationService struct {
	db            *gorm.DB
	mailer        *utils.Mailer
	escalationRepo *repository.TicketEscalationRepository
}

// NewSLAEscalationService creates a new SLA escalation service
func NewSLAEscalationService(db *gorm.DB, mailer *utils.Mailer) *SLAEscalationService {
	return &SLAEscalationService{
		db:            db,
		mailer:        mailer,
		escalationRepo: repository.NewTicketEscalationRepository(db),
	}
}

// SLAStatus represents the current SLA status of a ticket
type SLAStatus struct {
	Status         string // "OnTrack", "Warning", "Critical", "Breached"
	RemainingHours int
	ElapsedHours   int
	TotalSLAHours  int
	IsBreached     bool
	HoursOverdue   int
}

// GetSLAForPriority returns SLA hours based on priority
func GetSLAForPriority(priority string) int {
	switch priority {
	case "Low":
		return 10 * 24 // 10 days
	case "Standard":
		return 5 * 24 // 5 days
	case "Critical":
		return 2 * 24 // 2 days
	default:
		return 10 * 24 // Default to 10 days
	}
}

// CheckTicketSLA checks a single ticket for SLA violations
func (s *SLAEscalationService) CheckTicketSLA(ctx context.Context, ticket *models.Ticket) (*SLAStatus, error) {
	slaHours := GetSLAForPriority(string(ticket.Priority))

	createdTime := ticket.CreatedAt
	currentTime := time.Now()

	elapsedTime := currentTime.Sub(createdTime)
	elapsedHours := int(elapsedTime.Hours())

	remainingHours := slaHours - elapsedHours

	status := "OnTrack"
	isBreached := false
	hoursOverdue := 0

	if remainingHours <= 0 {
		status = "Breached"
		isBreached = true
		hoursOverdue = -remainingHours
	} else if remainingHours <= slaHours/4 {
		status = "Critical"
	} else if remainingHours <= slaHours/2 {
		status = "Warning"
	}

	return &SLAStatus{
		Status:         status,
		RemainingHours: remainingHours,
		ElapsedHours:   elapsedHours,
		TotalSLAHours:  slaHours,
		IsBreached:     isBreached,
		HoursOverdue:   hoursOverdue,
	}, nil
}

// CheckAndEscalateTickets checks all open tickets for SLA violations and sends notifications
func (s *SLAEscalationService) CheckAndEscalateTickets(ctx context.Context) error {
	// Get all open/in-progress tickets
	var tickets []models.Ticket

	if err := s.db.WithContext(ctx).
		Where("status != ?", "Closed").
		Preload("SupportEngineer").
		Preload("Customer").
		Find(&tickets).Error; err != nil {
		log.Printf("[SLA_CHECK_ERROR] Failed to fetch tickets: %v", err)
		return err
	}

	log.Printf("[SLA_CHECK] Checking %d tickets for SLA violations", len(tickets))

	for _, ticket := range tickets {
		slaStatus, err := s.CheckTicketSLA(ctx, &ticket)
		if err != nil {
			log.Printf("[SLA_CHECK_ERROR] Error checking SLA for ticket %s: %v", ticket.ID, err)
			continue
		}

		// Send breach notification if breached
		if slaStatus.IsBreached {
			if err := s.notifySLABreach(ctx, &ticket, slaStatus); err != nil {
				log.Printf("[SLA_NOTIFICATION_ERROR] Failed to notify SLA breach for ticket %s: %v", ticket.ID, err)
				// Don't return, continue with other tickets
			}
		}
	}

	return nil
}

// notifySLABreach sends email notifications for SLA breach
func (s *SLAEscalationService) notifySLABreach(ctx context.Context, ticket *models.Ticket, slaStatus *SLAStatus) error {
	if s.mailer == nil {
		log.Printf("[SLA_NOTIFICATION] Mailer not configured, skipping email notification for ticket %s", ticket.ID)
		return nil
	}

	// Check if escalation has already been sent for this ticket
	alreadyEscalated, err := s.escalationRepo.AlreadyEscalated(ticket.ID)
	if err != nil {
		log.Printf("[SLA_CHECK_ERROR] Failed to check escalation status for ticket %s: %v", ticket.ID, err)
		return err
	}

	// If already escalated, skip sending notification
	if alreadyEscalated {
		log.Printf("[SLA_NOTIFICATION_SKIP] Escalation already sent for ticket %s, skipping duplicate notification", ticket.ID)
		return nil
	}

	// Get support engineer and their user info
	var engineer models.SupportEngineer
	var engineerUser models.User
	if ticket.EngineerID != nil {
		if err := s.db.WithContext(ctx).Where("id = ?", ticket.EngineerID).First(&engineer).Error; err != nil {
			log.Printf("[SLA_NOTIFICATION_WARN] Failed to get engineer for ticket %s: %v", ticket.ID, err)
		} else {
			// Get engineer's user info for email
			if err := s.db.WithContext(ctx).Where("id = ?", engineer.UserID).First(&engineerUser).Error; err != nil {
				log.Printf("[SLA_NOTIFICATION_WARN] Failed to get engineer user info for ticket %s: %v", ticket.ID, err)
			}
		}
	}

	// Get admin (created_by user)
	var admin models.User
	if err := s.db.WithContext(ctx).Where("id = ?", ticket.CreatedBy).First(&admin).Error; err != nil {
		log.Printf("[SLA_NOTIFICATION_WARN] Failed to get admin for ticket %s: %v", ticket.ID, err)
	}

	// Get dashboard URL from environment or use default
	dashboardURL := "http://localhost:5173/support-reports" // Default, should be from config

	// Use ticket ID directly (already in VS/MM/YY/number format)
	ticketIDStr := ticket.ID

	// Get customer name
	customerName := "N/A"
	if ticket.Customer.Name != "" {
		customerName = ticket.Customer.Name
	}

	// Send email to support engineer
	if engineerUser.Email != "" {
		subject := fmt.Sprintf("🚨 SLA Breach Alert - Ticket %s", ticketIDStr)
		htmlContent := utils.SLABreachEmailTemplate(
			engineerUser.Name,
			ticketIDStr,
			customerName,
			ticket.Title,
			string(ticket.Priority),
			slaStatus.HoursOverdue,
			slaStatus.TotalSLAHours,
			dashboardURL,
		)

		if err := s.mailer.Send(engineerUser.Email, subject, htmlContent); err != nil {
			log.Printf("[SLA_NOTIFICATION_ERROR] Failed to send email to engineer %s: %v", engineerUser.Email, err)
			// Don't return error, try to send to admin anyway
		} else {
			log.Printf("[SLA_NOTIFICATION_SENT] Email sent to engineer %s for ticket %s", engineerUser.Email, ticket.ID)
		}
	}

	// Send email to admin
	if admin.Email != "" {
		subject := fmt.Sprintf("🚨 SLA Breach Alert - Ticket %s (Admin Notification)", ticketIDStr)
		htmlContent := utils.SLABreachEmailTemplate(
			admin.Name,
			ticketIDStr,
			customerName,
			ticket.Title,
			string(ticket.Priority),
			slaStatus.HoursOverdue,
			slaStatus.TotalSLAHours,
			dashboardURL,
		)

		if err := s.mailer.Send(admin.Email, subject, htmlContent); err != nil {
			log.Printf("[SLA_NOTIFICATION_ERROR] Failed to send email to admin %s: %v", admin.Email, err)
		} else {
			log.Printf("[SLA_NOTIFICATION_SENT] Email sent to admin %s for ticket %s", admin.Email, ticket.ID)
		}
	}

	// Create escalation record to prevent duplicate notifications
	if err := s.escalationRepo.Create(ticket.ID); err != nil {
		log.Printf("[SLA_ESCALATION_ERROR] Failed to create escalation record for ticket %s: %v", ticket.ID, err)
		return err
	}

	log.Printf("[SLA_ESCALATION_CREATED] Escalation record created for ticket %s", ticket.ID)

	return nil
}

// GetSLAStatus returns the current SLA status for a ticket
func (s *SLAEscalationService) GetSLAStatus(ctx context.Context, ticketID string) (string, error) {
	// Get ticket and check SLA status
	return "active", nil
}

// EscalateTicket escalates a ticket due to SLA violation
func (s *SLAEscalationService) EscalateTicket(ctx context.Context, ticketID string) error {
	return s.db.WithContext(ctx).
		Model(map[string]interface{}{}).
		Where("id = ?", ticketID).
		Update("status", "Escalated").
		Error
}
