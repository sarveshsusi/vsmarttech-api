package models

import (
	"time"

	"github.com/google/uuid"
)

/* =========================
   ENUMS
========================= */

type TicketStatus string

const (
	StatusOpen       TicketStatus = "Open"
	StatusAssigned   TicketStatus = "Assigned"
	StatusInProgress TicketStatus = "In Progress"
	StatusClosed     TicketStatus = "Closed"
)

type TicketPriority string

const (
	PriorityLow      TicketPriority = "Low"
	PriorityStandard TicketPriority = "Standard"
	PriorityCritical TicketPriority = "Critical"
)

type SupportMode string

const (
	SupportModeOnSite SupportMode = "On-site"
	SupportModeRemote SupportMode = "Remote"
	SupportModePhone  SupportMode = "Phone"
)

type ServiceCallType string

const (
	ServiceTypeWarranty ServiceCallType = "Warranty"
	ServiceTypeService  ServiceCallType = "Service"
	ServiceTypeAMC      ServiceCallType = "AMC"
)

/* =========================
   TICKET
========================= */

type Ticket struct {
	ID string `gorm:"type:varchar(20);primaryKey" json:"id"`

	CustomerID uuid.UUID `gorm:"type:uuid;index" json:"customer_id"`
	Customer   Customer  `gorm:"foreignKey:CustomerID" json:"customer"`

	CustomerSolutionID *uuid.UUID        `gorm:"type:uuid;index" json:"customer_solution_id,omitempty"`
	CustomerSolution   *CustomerSolution `gorm:"foreignKey:CustomerSolutionID" json:"customer_solution,omitempty"`

	// ✅ ASSIGNED ENGINEER (SNAPSHOT)
	EngineerID      *uuid.UUID       `gorm:"type:uuid;index" json:"engineer_id,omitempty"`
	SupportEngineer *SupportEngineer `gorm:"foreignKey:EngineerID" json:"support_engineer,omitempty"`

	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      TicketStatus `json:"status"`

	Priority        TicketPriority  `json:"priority"`
	SupportMode     SupportMode     `json:"support_mode"`
	ServiceCallType ServiceCallType `json:"service_call_type"`

	SLAHours int        `json:"sla_hours"`
	TargetAt *time.Time `json:"target_at"`

	ClosureProofImage *string    `json:"closure_proof_image,omitempty"`
	SupportComment    *string    `json:"support_comment,omitempty"`
	ClosedAt          *time.Time `json:"closed_at,omitempty"`

	// Attachments relationship
	Attachments []TicketAttachment `gorm:"foreignKey:TicketID" json:"attachments,omitempty"`

	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Ticket) TableName() string {
	return "tickets"
}

/* =========================
   ASSIGNMENT
========================= */

type TicketAssignment struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID           string    `json:"ticket_id" gorm:"type:varchar(20);index"`
	CustomerSolutionID uuid.UUID `json:"customer_solution_id" gorm:"type:uuid;index"`
	EngineerID         uuid.UUID `json:"engineer_id" gorm:"type:uuid;index"`
	AssignedBy         uuid.UUID `json:"assigned_by" gorm:"type:uuid"`
	AssignedAt         time.Time `json:"assigned_at"`
}

func (TicketAssignment) TableName() string {
	return "ticket_assignments"
}

/* =========================
   STATUS HISTORY
========================= */

type TicketStatusHistory struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID  string    `json:"ticket_id" gorm:"type:varchar(20);index"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	ChangedBy uuid.UUID `json:"changed_by" gorm:"type:uuid"`
	ChangedAt time.Time `json:"changed_at"`
}

func (TicketStatusHistory) TableName() string {
	return "ticket_status_histories"
}

/* =========================
   COMMENTS
========================= */

type TicketComment struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID   string    `json:"ticket_id" gorm:"type:varchar(20);index"`
	UserID     uuid.UUID `json:"user_id" gorm:"type:uuid"`
	Comment    string    `json:"comment" gorm:"type:text"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

func (TicketComment) TableName() string {
	return "ticket_comments"
}

/* =========================
   ATTACHMENTS
========================= */

type TicketAttachment struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID   string    `json:"ticket_id" gorm:"type:varchar(20);index"`
	FileURL    string    `json:"file_url"`
	FileName   string    `json:"file_name"`
	FileType   string    `json:"file_type"`
	UploadedBy uuid.UUID `json:"uploaded_by" gorm:"type:uuid"`
	CreatedAt  time.Time `json:"created_at"`
}

func (TicketAttachment) TableName() string {
	return "ticket_attachments"
}

/* =========================
   FEEDBACK
========================= */

type TicketFeedback struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID   string    `json:"ticket_id" gorm:"type:varchar(20)"`
	EngineerID uuid.UUID `json:"engineer_id" gorm:"type:uuid"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

func (TicketFeedback) TableName() string {
	return "ticket_feedbacks"
}
