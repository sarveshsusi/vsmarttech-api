package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"rbac/models"
	"rbac/repository"
	"rbac/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationService struct {
	db           *gorm.DB
	notifRepo    *repository.NotificationRepository
	ticketRepo   *repository.TicketRepository
	userRepo     *repository.UserRepository
	customerRepo *repository.CustomerRepository
	mailer       *utils.Mailer
	frontendURL  string
}

func NewNotificationService(
	db *gorm.DB,
	notifRepo *repository.NotificationRepository,
	ticketRepo *repository.TicketRepository,
	userRepo *repository.UserRepository,
	customerRepo *repository.CustomerRepository,
	mailer *utils.Mailer,
	frontendURL string,
) *NotificationService {
	return &NotificationService{
		db:           db,
		notifRepo:    notifRepo,
		ticketRepo:   ticketRepo,
		userRepo:     userRepo,
		customerRepo: customerRepo,
		mailer:       mailer,
		frontendURL:  strings.TrimRight(frontendURL, "/"),
	}
}

func (s *NotificationService) customerTicketURL(ticketID string) string {
	base := s.frontendURL
	if base == "" {
		base = "http://localhost:5173"
	}
	return base + "/customer/tickets"
}

func (s *NotificationService) adminTicketURL(ticketID string) string {
	base := s.frontendURL
	if base == "" {
		base = "http://localhost:5173"
	}
	return base + "/admin/tickets/details?id=" + url.QueryEscape(ticketID)
}

func (s *NotificationService) resolveEngineerDisplayName(ticket *models.Ticket) string {
	if ticket == nil {
		return "Support Team"
	}
	if ticket.SupportEngineer != nil {
		if ticket.SupportEngineer.User.Name != "" {
			return ticket.SupportEngineer.User.Name
		}
		if ticket.SupportEngineer.User.Email != "" {
			return ticket.SupportEngineer.User.Email
		}
	}
	if ticket.EngineerID == nil {
		return "Support Team"
	}
	var eng models.SupportEngineer
	if err := s.db.Preload("User").Where("id = ?", *ticket.EngineerID).First(&eng).Error; err == nil {
		if eng.User.Name != "" {
			return eng.User.Name
		}
		if eng.User.Email != "" {
			return eng.User.Email
		}
	}
	return "Support Team"
}

/* =========================
   TRIGGER NOTIFICATION
========================= */

type NotificationPayload struct {
	TicketID  string    `json:"ticket_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	OldStatus *string   `json:"old_status,omitempty"`
	NewStatus *string   `json:"new_status,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (s *NotificationService) CreateTicketNotification(
	userID uuid.UUID,
	ticketID string,
	notificationType models.NotificationType,
	title string,
	message string,
	oldStatus *string,
	newStatus *string,
) error {
	notification := &models.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		TicketID:  &ticketID,
		Type:      notificationType,
		Title:     title,
		Message:   message,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		IsRead:    false,
		Metadata:  "{}",
	}

	// Save notification to database
	if err := s.notifRepo.Create(notification); err != nil {
		log.Printf("[NOTIFICATION_ERROR] Failed to create notification: %v", err)
		return err
	}

	// Create webhook event for real-time delivery
	payload := NotificationPayload{
		TicketID:  ticketID,
		Type:      string(notificationType),
		Title:     title,
		Message:   message,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Timestamp: time.Now(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[WEBHOOK_ERROR] Failed to marshal payload: %v", err)
		return err
	}

	webhookEvent := &models.WebhookEvent{
		ID:        uuid.New(),
		EventType: string(notificationType),
		TicketID:  ticketID,
		Payload:   string(payloadJSON),
	}

	if err := s.notifRepo.CreateWebhookEvent(webhookEvent); err != nil {
		log.Printf("[WEBHOOK_ERROR] Failed to create webhook event: %v", err)
		return err
	}

	log.Printf(
		"[NOTIFICATION_CREATED] user_id=%s ticket_id=%s type=%s",
		userID,
		ticketID,
		notificationType,
	)

	return nil
}

/* =========================
   TICKET CREATED NOTIFICATION
========================= */

func (s *NotificationService) NotifyTicketCreated(
	ticketID string,
	customerID uuid.UUID,
) error {
	log.Printf("[NOTIFY_TICKET_CREATED] Starting for ticketID=%s customerID=%s", ticketID, customerID)

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		log.Printf("[NOTIFY_TICKET_CREATED_ERROR] Failed to get ticket: %v", err)
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	log.Printf("[NOTIFY_TICKET_CREATED] Got ticket: %s", ticket.Title)

	// Get customer to find their UserID
	var customer models.Customer
	err = s.db.Where("id = ?", customerID).First(&customer).Error
	if err != nil {
		log.Printf("[NOTIFY_TICKET_CREATED_WARN] Failed to get customer: %v", err)
	}

	// Notify admins about new ticket
	adminUsers, err := s.userRepo.GetUsersByRole(models.RoleAdmin)
	if err != nil {
		log.Printf("[NOTIFY_TICKET_CREATED_WARN] Failed to get admin users: %v", err)
	} else {
		log.Printf("[NOTIFY_TICKET_CREATED] Found %d admins", len(adminUsers))
		for _, admin := range adminUsers {
			adminMsg := fmt.Sprintf(
				"New ticket created: '%s' by customer",
				ticket.Title,
			)
			if err := s.CreateTicketNotification(
				admin.ID,
				ticketID,
				models.NotificationTypeTicketCreated,
				"New Ticket Created",
				adminMsg,
				nil,
				nil,
			); err != nil {
				log.Printf("[NOTIFY_TICKET_CREATED_ERROR] Failed to create admin notification for %s: %v", admin.ID, err)
			}
		}
	}

	// Notify customer about ticket creation confirmation + email
	if err == nil && customer.UserID != uuid.Nil {
		customerMsg := fmt.Sprintf(
			"Your ticket '%s' has been successfully created and is awaiting assignment",
			ticket.Title,
		)
		if err := s.CreateTicketNotification(
			customer.UserID,
			ticketID,
			models.NotificationTypeTicketCreated,
			"Ticket Created",
			customerMsg,
			nil,
			nil,
		); err != nil {
			log.Printf("[NOTIFY_TICKET_CREATED_ERROR] Failed to create customer notification: %v", err)
		}

		var user models.User
		if uErr := s.db.Where("id = ?", customer.UserID).First(&user).Error; uErr == nil && user.Email != "" {
			name := user.Name
			if name == "" {
				name = customer.Name
			}
			if name == "" {
				name = "Customer"
			}
			emailHTML := utils.TicketCreatedEmailTemplate(
				name,
				ticket.ID,
				ticket.Title,
				s.customerTicketURL(ticket.ID),
			)
			subject := "We received your support ticket #" + ticket.ID
			if s.mailer != nil {
				if sendErr := s.mailer.Send(user.Email, subject, emailHTML); sendErr != nil {
					log.Printf("[EMAIL_ERROR] Failed to send ticket-created email to %s: %v", user.Email, sendErr)
				} else {
					log.Printf("[EMAIL_SENT] Ticket created email sent to %s", user.Email)
				}
			} else {
				log.Printf("[MAILER_WARN] Mailer not configured, skipping ticket-created email")
			}
		}
	} else {
		log.Printf("[NOTIFY_TICKET_CREATED_WARN] Customer not found or no UserID, skipping customer notification")
	}

	log.Printf("[NOTIFY_TICKET_CREATED_SUCCESS] Notifications created for ticketID=%s", ticketID)
	return nil
}

/* =========================
   TICKET STATUS CHANGED NOTIFICATION
========================= */

func (s *NotificationService) NotifyTicketStatusChanged(
	ticketID string,
	oldStatus string,
	newStatus string,
	changedByUserID uuid.UUID,
) error {
	log.Printf("[NOTIFY_STATUS_CHANGED] Starting for ticketID=%s oldStatus=%s newStatus=%s", ticketID, oldStatus, newStatus)

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		log.Printf("[NOTIFY_STATUS_CHANGED_ERROR] Failed to get ticket: %v", err)
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// Get customer to find their UserID
	var customer models.Customer
	err = s.db.Where("id = ?", ticket.CustomerID).First(&customer).Error
	if err != nil {
		log.Printf("[NOTIFY_STATUS_CHANGED_WARN] Failed to get customer: %v", err)
	}

	// Notify customer about status change
	if err == nil && customer.UserID != uuid.Nil {
		message := fmt.Sprintf(
			"Your ticket '%s' status has changed from %s to %s",
			ticket.Title,
			oldStatus,
			newStatus,
		)

		if err := s.CreateTicketNotification(
			customer.UserID,
			ticketID,
			models.NotificationTypeTicketStatusChanged,
			"Ticket Status Updated",
			message,
			&oldStatus,
			&newStatus,
		); err != nil {
			log.Printf("[NOTIFY_STATUS_CHANGED_ERROR] Failed to notify customer: %v", err)
		} else {
			log.Printf("[NOTIFICATION_CREATED] customer user_id=%s ticket_id=%s type=ticket_status_changed", customer.UserID, ticketID)
		}
	}

	// Notify support engineer who performed the action
	if changedByUserID != uuid.Nil {
		engineerMsg := fmt.Sprintf(
			"You started working on ticket '%s' - status changed from %s to %s",
			ticket.Title,
			oldStatus,
			newStatus,
		)
		if err := s.CreateTicketNotification(
			changedByUserID,
			ticketID,
			models.NotificationTypeTicketStatusChanged,
			"Ticket Status Changed",
			engineerMsg,
			&oldStatus,
			&newStatus,
		); err != nil {
			log.Printf("[NOTIFY_STATUS_CHANGED_ERROR] Failed to notify engineer: %v", err)
		} else {
			log.Printf("[NOTIFICATION_CREATED] engineer user_id=%s ticket_id=%s type=ticket_status_changed", changedByUserID, ticketID)
		}
	}

	// Notify support engineer (if assigned) about status change - only if different from changedBy
	if ticket.EngineerID != nil && *ticket.EngineerID != uuid.Nil && *ticket.EngineerID != changedByUserID {
		engineer, err := s.userRepo.GetByID(*ticket.EngineerID)
		if err == nil && engineer != nil {
			engineerMsg := fmt.Sprintf(
				"Ticket '%s' status changed from %s to %s",
				ticket.Title,
				oldStatus,
				newStatus,
			)
			if err := s.CreateTicketNotification(
				*ticket.EngineerID,
				ticketID,
				models.NotificationTypeTicketStatusChanged,
				"Ticket Status Changed",
				engineerMsg,
				&oldStatus,
				&newStatus,
			); err != nil {
				log.Printf("[NOTIFY_STATUS_CHANGED_ERROR] Failed to notify engineer: %v", err)
			} else {
				log.Printf("[NOTIFICATION_CREATED] engineer user_id=%s ticket_id=%s type=ticket_status_changed", ticket.EngineerID, ticketID)
			}
		}
	}

	// Notify all admins about the change
	adminUsers, err := s.userRepo.GetUsersByRole(models.RoleAdmin)
	if err == nil && len(adminUsers) > 0 {
		log.Printf("[NOTIFY_STATUS_CHANGED] Found %d admins to notify", len(adminUsers))
		for _, admin := range adminUsers {
			adminMsg := fmt.Sprintf(
				"Ticket '%s' status changed from %s to %s",
				ticket.Title,
				oldStatus,
				newStatus,
			)
			if err := s.CreateTicketNotification(
				admin.ID,
				ticketID,
				models.NotificationTypeTicketStatusChanged,
				"Ticket Status Changed",
				adminMsg,
				&oldStatus,
				&newStatus,
			); err != nil {
				log.Printf("[NOTIFY_STATUS_CHANGED_ERROR] Failed to notify admin %s: %v", admin.ID, err)
			} else {
				log.Printf("[NOTIFICATION_CREATED] admin user_id=%s ticket_id=%s type=ticket_status_changed", admin.ID, ticketID)
			}
		}
	}

	log.Printf("[NOTIFY_STATUS_CHANGED_SUCCESS] Notifications sent for ticketID=%s", ticketID)
	return nil
}

/* =========================
   TICKET ASSIGNED NOTIFICATION
========================= */

func (s *NotificationService) NotifyTicketAssigned(
	ticketID string,
	engineerID uuid.UUID,
) error {
	log.Printf("[NOTIFY_ASSIGNED] Starting for ticketID=%s engineerID=%s", ticketID, engineerID)

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		log.Printf("[NOTIFY_ASSIGNED_ERROR] Failed to get ticket: %v", err)
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// Try to get engineer details, but don't fail if not found
	engineerName := "a support engineer"
	if eng, err := s.userRepo.GetByID(engineerID); err != nil {
		log.Printf("[NOTIFY_ASSIGNED_WARN] Failed to get engineer details: %v (will proceed with basic notification)", err)
	} else if eng != nil {
		engineerName = eng.Name
	}

	// Build customer name
	customerName := "a customer"
	if ticket.Customer.Name != "" {
		customerName = ticket.Customer.Name
	}

	// Notify engineer about assignment
	engineerMsg := fmt.Sprintf(
		"You have been assigned to ticket '%s' from %s",
		ticket.Title,
		customerName,
	)
	if err := s.CreateTicketNotification(
		engineerID,
		ticketID,
		models.NotificationTypeTicketAssigned,
		"New Ticket Assigned",
		engineerMsg,
		nil,
		nil,
	); err != nil {
		log.Printf("[NOTIFY_ASSIGNED_ERROR] Failed to notify engineer: %v", err)
		return err
	} else {
		log.Printf("[NOTIFICATION_CREATED] engineer user_id=%s ticket_id=%s type=ticket_assigned", engineerID, ticketID)
	}

	// Get customer to find their UserID
	var customer models.Customer
	err = s.db.Where("id = ?", ticket.CustomerID).First(&customer).Error
	if err != nil {
		log.Printf("[NOTIFY_ASSIGNED_WARN] Failed to get customer: %v", err)
	}

	// Notify customer about assignment
	if err == nil && customer.UserID != uuid.Nil {
		customerMsg := fmt.Sprintf(
			"Your ticket '%s' has been assigned to %s",
			ticket.Title,
			engineerName,
		)
		if err := s.CreateTicketNotification(
			customer.UserID,
			ticketID,
			models.NotificationTypeTicketAssigned,
			"Ticket Assigned",
			customerMsg,
			nil,
			nil,
		); err != nil {
			log.Printf("[NOTIFY_ASSIGNED_ERROR] Failed to notify customer: %v", err)
		} else {
			log.Printf("[NOTIFICATION_CREATED] customer user_id=%s ticket_id=%s type=ticket_assigned", customer.UserID, ticketID)
		}
	}

	// Notify admin
	adminUsers, err := s.userRepo.GetUsersByRole(models.RoleAdmin)
	if err == nil && len(adminUsers) > 0 {
		log.Printf("[NOTIFY_ASSIGNED] Found %d admins to notify", len(adminUsers))
		for _, admin := range adminUsers {
			adminMsg := fmt.Sprintf(
				"Ticket '%s' assigned to %s",
				ticket.Title,
				engineerName,
			)
			if err := s.CreateTicketNotification(
				admin.ID,
				ticketID,
				models.NotificationTypeTicketAssigned,
				"Ticket Assigned",
				adminMsg,
				nil,
				nil,
			); err != nil {
				log.Printf("[NOTIFY_ASSIGNED_ERROR] Failed to notify admin %s: %v", admin.ID, err)
			} else {
				log.Printf("[NOTIFICATION_CREATED] admin user_id=%s ticket_id=%s type=ticket_assigned", admin.ID, ticketID)
			}
		}
	}

	log.Printf("[NOTIFY_ASSIGNED_SUCCESS] Notifications sent for ticketID=%s", ticketID)
	return nil
}

/* =========================
   TICKET CLOSED NOTIFICATION
========================= */

func (s *NotificationService) NotifyTicketClosed(
	ticketID string,
	supportComment string,
) error {
	log.Printf("[NOTIFY_CLOSED] Starting for ticketID=%s", ticketID)

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		log.Printf("[NOTIFY_CLOSED_ERROR] Failed to get ticket: %v", err)
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	// Get customer to find their UserID and email
	var customer models.Customer
	var user models.User
	err = s.db.Where("id = ?", ticket.CustomerID).First(&customer).Error
	if err != nil {
		log.Printf("[NOTIFY_CLOSED_WARN] Failed to get customer: %v", err)
	}

	// Get the engineer name if available
	engineerName := s.resolveEngineerDisplayName(ticket)

	// Notify customer about closure
	if err == nil && customer.UserID != uuid.Nil {
		// Get customer user to get email
		err = s.db.Where("id = ?", customer.UserID).First(&user).Error
		if err == nil && user.Email != "" {
			log.Printf("[NOTIFY_CLOSED] Sending closure email to customer %s at %s", user.Name, user.Email)
			closureDate := time.Now().Format("02 Jan 2006, 3:04 PM")
			name := user.Name
			if name == "" {
				name = customer.Name
			}
			if name == "" {
				name = "Customer"
			}
			emailHTML := utils.TicketClosureEmailTemplate(
				name,
				ticket.ID,
				ticket.Title,
				engineerName,
				closureDate,
				supportComment,
				s.customerTicketURL(ticket.ID),
			)

			if s.mailer != nil {
				emailSubject := "Your support ticket #" + ticket.ID + " has been resolved"
				if sendErr := s.mailer.Send(user.Email, emailSubject, emailHTML); sendErr != nil {
					log.Printf("[EMAIL_ERROR] Failed to send closure email to customer %s: %v", user.Email, sendErr)
				} else {
					log.Printf("[EMAIL_SENT] Closure email sent to customer %s", user.Email)
				}
			} else {
				log.Printf("[MAILER_WARN] Mailer not configured, skipping email send")
			}
		} else if err != nil {
			log.Printf("[NOTIFY_CLOSED_WARN] Failed to get customer user email: %v", err)
		} else {
			log.Printf("[NOTIFY_CLOSED_WARN] Customer email is empty for user_id=%s", customer.UserID)
		}

		// Create in-app notification for customer
		customerMsg := fmt.Sprintf(
			"Your ticket '%s' has been resolved. Resolved by: %s. Comment: %s",
			ticket.Title,
			engineerName,
			supportComment,
		)
		if err := s.CreateTicketNotification(
			customer.UserID,
			ticketID,
			models.NotificationTypeTicketClosed,
			"Ticket Closed",
			customerMsg,
			nil,
			nil,
		); err != nil {
			log.Printf("[NOTIFY_CLOSED_ERROR] Failed to notify customer: %v", err)
		} else {
			log.Printf("[NOTIFICATION_CREATED] customer user_id=%s ticket_id=%s type=ticket_closed", customer.UserID, ticketID)
		}
	}

	// Notify support engineer (if assigned) about closure
	if ticket.EngineerID != nil && *ticket.EngineerID != uuid.Nil {
		var eng models.SupportEngineer
		if err := s.db.Where("id = ?", *ticket.EngineerID).First(&eng).Error; err == nil && eng.UserID != uuid.Nil {
			engineerMsg := fmt.Sprintf(
				"Ticket '%s' has been closed. Support comment: %s",
				ticket.Title,
				supportComment,
			)
			if err := s.CreateTicketNotification(
				eng.UserID,
				ticketID,
				models.NotificationTypeTicketClosed,
				"Ticket Closed",
				engineerMsg,
				nil,
				nil,
			); err != nil {
				log.Printf("[NOTIFY_CLOSED_ERROR] Failed to notify engineer: %v", err)
			} else {
				log.Printf("[NOTIFICATION_CREATED] engineer user_id=%s ticket_id=%s type=ticket_closed", eng.UserID, ticketID)
			}
		}
	}

	// Notify admin about closure
	adminUsers, err := s.userRepo.GetUsersByRole(models.RoleAdmin)
	if err == nil && len(adminUsers) > 0 {
		log.Printf("[NOTIFY_CLOSED] Found %d admins to notify", len(adminUsers))
		for _, admin := range adminUsers {
			adminMsg := fmt.Sprintf(
				"Ticket '%s' has been closed",
				ticket.Title,
			)
			if err := s.CreateTicketNotification(
				admin.ID,
				ticketID,
				models.NotificationTypeTicketClosed,
				"Ticket Closed",
				adminMsg,
				nil,
				nil,
			); err != nil {
				log.Printf("[NOTIFY_CLOSED_ERROR] Failed to notify admin %s: %v", admin.ID, err)
			} else {
				log.Printf("[NOTIFICATION_CREATED] admin user_id=%s ticket_id=%s type=ticket_closed", admin.ID, ticketID)
			}
		}
	}

	log.Printf("[NOTIFY_CLOSED_SUCCESS] Notifications sent for ticketID=%s", ticketID)
	return nil
}

/* =========================
   GET USER NOTIFICATIONS
========================= */

func (s *NotificationService) GetUserNotifications(userID uuid.UUID, page int, pageSize int) ([]models.Notification, int64, error) {
	offset := (page - 1) * pageSize
	notifications, err := s.notifRepo.GetUserNotifications(userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	// Ensure we never return nil, always return empty slice
	if notifications == nil {
		notifications = []models.Notification{}
	}

	// Get total count
	count, err := s.getNotificationCount(userID)
	if err != nil {
		return nil, 0, err
	}

	return notifications, count, nil
}

func (s *NotificationService) getNotificationCount(userID uuid.UUID) (int64, error) {
	var count int64
	return count, s.db.
		Model(&models.Notification{}).
		Where("user_id = ?", userID).
		Count(&count).Error
}

/* =========================
   GET UNREAD COUNT
========================= */

func (s *NotificationService) GetUnreadCount(userID uuid.UUID) (int64, error) {
	return s.notifRepo.GetUnreadCount(userID)
}

/* =========================
   MARK AS READ
========================= */

func (s *NotificationService) MarkAsRead(notificationID uuid.UUID) error {
	return s.notifRepo.MarkAsRead(notificationID)
}

/* =========================
   MARK ALL AS READ
========================= */

func (s *NotificationService) MarkAllAsRead(userID uuid.UUID) error {
	return s.notifRepo.MarkAllAsRead(userID)
}

/* =========================
   GET NOTIFICATION PREFERENCE
========================= */

func (s *NotificationService) GetPreference(userID uuid.UUID) (*models.NotificationPreference, error) {
	return s.notifRepo.GetPreference(userID)
}

/* =========================
   UPDATE NOTIFICATION PREFERENCE
========================= */

func (s *NotificationService) UpdatePreference(userID uuid.UUID, updates map[string]interface{}) error {
	return s.notifRepo.UpdatePreference(userID, updates)
}

/* =========================
   AMC NOTIFICATIONS
========================= */

// NotifyAMCAssigned sends notification when AMC is assigned to engineer
func (s *NotificationService) NotifyAMCAssigned(engineerID uuid.UUID, assignmentID uuid.UUID, startDate, endDate time.Time) {
	go func() {
		title := "AMC Assignment"
		message := fmt.Sprintf("You have been assigned an AMC contract from %s to %s. Please check your AMC dashboard for details.",
			startDate.Format("02-Jan-2006"),
			endDate.Format("02-Jan-2006"),
		)

		payload := map[string]interface{}{
			"type":          "amc_assigned",
			"assignment_id": assignmentID.String(),
			"start_date":    startDate,
			"end_date":      endDate,
		}

		payloadJSON, _ := json.Marshal(payload)

		notification := &models.Notification{
			UserID:   engineerID,
			Title:    title,
			Message:  message,
			Type:     models.NotificationType("amc_assigned"),
			Metadata: string(payloadJSON),
			IsRead:   false,
		}

		s.notifRepo.Create(notification)
	}()
}

// NotifyVisitApproaching sends notification when visit date is approaching
func (s *NotificationService) NotifyVisitApproaching(engineerID uuid.UUID, visitID uuid.UUID, visitDate time.Time) {
	go func() {
		title := "AMC Visit Approaching"
		message := fmt.Sprintf("Your scheduled AMC site visit is on %s. Please complete this visit and upload proof.",
			visitDate.Format("02-Jan-2006"),
		)

		payload := map[string]interface{}{
			"type":     "visit_approaching",
			"visit_id": visitID.String(),
			"date":     visitDate,
		}

		payloadJSON, _ := json.Marshal(payload)

		notification := &models.Notification{
			UserID:   engineerID,
			Title:    title,
			Message:  message,
			Type:     models.NotificationType("visit_approaching"),
			Metadata: string(payloadJSON),
			IsRead:   false,
		}

		s.notifRepo.Create(notification)
	}()
}

// NotifyVisitCompleted sends notification to admin when visit is completed
func (s *NotificationService) NotifyVisitCompleted(assignmentID uuid.UUID, visitID uuid.UUID, visitDate time.Time) {
	go func() {
		// Get assignment details to find admin
		// For now, notify the assignment creator (admin)
		title := "AMC Visit Completed"
		message := fmt.Sprintf("An AMC site visit was completed on %s. Check the proof and update in the dashboard.",
			visitDate.Format("02-Jan-2006"),
		)

		payload := map[string]interface{}{
			"type":          "visit_completed",
			"assignment_id": assignmentID.String(),
			"visit_id":      visitID.String(),
			"visit_date":    visitDate,
		}

		payloadJSON, _ := json.Marshal(payload)

		// This would notify the admin (need to get admin ID from assignment)
		// For now, this is a placeholder
		log.Printf("Visit completed notification for assignment: %s, visit: %s", assignmentID, visitID)
		_ = title
		_ = message
		_ = payloadJSON
	}()
}

// NotifyVisitOverdue sends notification when visit is overdue
func (s *NotificationService) NotifyVisitOverdue(engineerID uuid.UUID, visitID uuid.UUID, quarterEndDate time.Time) {
	go func() {
		title := "⚠️ AMC Visit Overdue"
		message := fmt.Sprintf("Your AMC site visit was due by %s. Please complete it immediately and upload proof.",
			quarterEndDate.Format("02-Jan-2006"),
		)

		payload := map[string]interface{}{
			"type":     "visit_overdue",
			"visit_id": visitID.String(),
			"due_date": quarterEndDate,
		}

		payloadJSON, _ := json.Marshal(payload)

		notification := &models.Notification{
			UserID:   engineerID,
			Title:    title,
			Message:  message,
			Type:     models.NotificationType("visit_overdue"),
			Metadata: string(payloadJSON),
			IsRead:   false,
		}

		s.notifRepo.Create(notification)
	}()
}
