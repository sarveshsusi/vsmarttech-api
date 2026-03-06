package models

import (
	"time"

	"github.com/google/uuid"
)

/* =========================
   NOTIFICATION TYPES
========================= */

type NotificationType string

const (
	NotificationTypeTicketCreated       NotificationType = "ticket_created"
	NotificationTypeTicketAssigned      NotificationType = "ticket_assigned"
	NotificationTypeTicketStarted       NotificationType = "ticket_started"
	NotificationTypeTicketClosed        NotificationType = "ticket_closed"
	NotificationTypeTicketCommented     NotificationType = "ticket_commented"
	NotificationTypeTicketStatusChanged NotificationType = "ticket_status_changed"

	// AMC/Warranty expiry notifications (for customers)
	NotificationTypeAMCExpiry3Months      NotificationType = "amc_expiry_3_months"
	NotificationTypeAMCExpiry1Month       NotificationType = "amc_expiry_1_month"
	NotificationTypeAMCExpiry7Days        NotificationType = "amc_expiry_7_days"
	NotificationTypeWarrantyExpiry3Months NotificationType = "warranty_expiry_3_months"
	NotificationTypeWarrantyExpiry1Month  NotificationType = "warranty_expiry_1_month"
	NotificationTypeWarrantyExpiry7Days   NotificationType = "warranty_expiry_7_days"

	// Admin daily summary notifications
	NotificationTypeAdminExpiryAlert NotificationType = "admin_expiry_alert"
)

/* =========================
   NOTIFICATION
========================= */

type Notification struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	User   *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`

	TicketID *uuid.UUID `gorm:"type:uuid;index" json:"ticket_id,omitempty"`
	Ticket   *Ticket    `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`

	// For AMC/Warranty notifications
	CustomerSolutionID *uuid.UUID        `gorm:"type:uuid;index" json:"customer_solution_id,omitempty"`
	CustomerSolution   *CustomerSolution `gorm:"foreignKey:CustomerSolutionID" json:"customer_solution,omitempty"`

	Type    NotificationType `gorm:"type:varchar(50)" json:"type"`
	Title   string           `gorm:"type:varchar(255)" json:"title"`
	Message string           `gorm:"type:text" json:"message"`

	// Status info
	OldStatus *string `gorm:"type:varchar(50)" json:"old_status,omitempty"`
	NewStatus *string `gorm:"type:varchar(50)" json:"new_status,omitempty"`

	// Read status
	IsRead bool       `gorm:"default:false" json:"is_read"`
	ReadAt *time.Time `json:"read_at,omitempty"`

	// Metadata for rich notifications
	Metadata string `gorm:"type:jsonb" json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Notification) TableName() string {
	return "notifications"
}

/* =========================
   WEBHOOK EVENT
========================= */

type WebhookEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	EventType string    `gorm:"type:varchar(100);index" json:"event_type"`
	TicketID  uuid.UUID `gorm:"type:uuid;index" json:"ticket_id"`

	// Event payload
	Payload string `gorm:"type:jsonb" json:"payload"`

	// Delivery status
	IsDelivered  bool       `gorm:"default:false" json:"is_delivered"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	RetryCount   int        `gorm:"default:0" json:"retry_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WebhookEvent) TableName() string {
	return "webhook_events"
}

/* =========================
   NOTIFICATION PREFERENCES
========================= */

type NotificationPreference struct {
	ID     uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"user_id"`
	User   *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Notification settings
	EmailNotifications   bool `gorm:"default:true" json:"email_notifications"`
	InAppNotifications   bool `gorm:"default:true" json:"in_app_notifications"`
	WebhookNotifications bool `gorm:"default:true" json:"webhook_notifications"`

	// Notification types to receive
	TicketCreatedNotification      bool `gorm:"default:true" json:"ticket_created_notification"`
	TicketAssignedNotification     bool `gorm:"default:true" json:"ticket_assigned_notification"`
	TicketStatusChangeNotification bool `gorm:"default:true" json:"ticket_status_change_notification"`
	TicketClosedNotification       bool `gorm:"default:true" json:"ticket_closed_notification"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (NotificationPreference) TableName() string {
	return "notification_preferences"
}
