package models

import (
	"time"

	"github.com/google/uuid"
)

/*
	=========================
	  AMC ASSIGNMENT

=========================
*/
type AMCAssignment struct {
	ID                 uuid.UUID `json:"id" gorm:"primaryKey"`
	CustomerSolutionID uuid.UUID `json:"customer_solution_id"`
	SupportEngineerID  uuid.UUID `json:"support_engineer_id"`
	AssignedBy         uuid.UUID `json:"assigned_by"`
	AssignedAt         time.Time `json:"assigned_at"`
	AMCStartDate       time.Time `json:"amc_start_date"`
	AMCEndDate         time.Time `json:"amc_end_date"`
	Status             string    `json:"status"` // active, completed, expired, closed
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	// Relations
	CustomerSolution *CustomerSolution    `json:"customer_solution,omitempty" gorm:"foreignKey:CustomerSolutionID"`
	SupportEngineer  *SupportEngineer     `json:"support_engineer,omitempty" gorm:"foreignKey:SupportEngineerID"`
	Visits           []AMCVisit           `json:"visits,omitempty" gorm:"foreignKey:AMCAssignmentID"`
	AssignmentEvents []AMCAssignmentEvent `json:"assignment_events,omitempty" gorm:"foreignKey:AMCAssignmentID"`
}

/*
	=========================
	  AMC VISIT (Quarterly)

=========================
*/
type AMCVisit struct {
	ID                uuid.UUID  `json:"id" gorm:"primaryKey"`
	AMCAssignmentID   uuid.UUID  `json:"amc_assignment_id"`
	SupportEngineerID *uuid.UUID `json:"support_engineer_id,omitempty" gorm:"type:uuid;index"`
	QuarterStartDate  time.Time  `json:"quarter_start_date"` // e.g., 2024-01-01 for Q1
	QuarterEndDate    time.Time  `json:"quarter_end_date"`   // e.g., 2024-03-31 for Q1
	VisitScheduledFor time.Time  `json:"visit_scheduled_for"`
	VisitDate         *time.Time `json:"visit_date"` // When actually visited
	Status            string     `json:"status"`     // pending, completed, overdue
	Notes             string     `json:"notes"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CompletedAt       *time.Time `json:"completed_at"`

	// Relations
	AMCAssignment   *AMCAssignment   `json:"amc_assignment,omitempty" gorm:"foreignKey:AMCAssignmentID"`
	SupportEngineer *SupportEngineer `json:"support_engineer,omitempty" gorm:"foreignKey:SupportEngineerID"`
	Proofs          []AMCVisitProof  `json:"proofs,omitempty" gorm:"foreignKey:AMCVisitID"`
}

/*
	=========================
	  AMC VISIT PROOF (Images)

=========================
*/
type AMCVisitProof struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey"`
	AMCVisitID  uuid.UUID `json:"amc_visit_id"`
	ImagePath   string    `json:"image_path"`  // Local file path
	Description string    `json:"description"` // Description of proof
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	UploadedAt  time.Time `json:"uploaded_at"`

	// Relations
	AMCVisit *AMCVisit `json:"amc_visit,omitempty" gorm:"foreignKey:AMCVisitID"`
}

/*
	=========================
	  TABLE NAMES

=========================
*/
func (AMCAssignment) TableName() string {
	return "amc_assignments"
}

func (AMCVisit) TableName() string {
	return "amc_visits"
}

func (AMCVisitProof) TableName() string {
	return "amc_visit_proofs"
}

/*
	=========================
	  AMC ASSIGNMENT EVENTS

=========================
*/
const (
	AMCEventAssigned   = "assigned"
	AMCEventReassigned = "reassigned"
)

type AMCAssignmentEvent struct {
	ID              uuid.UUID  `json:"id" gorm:"primaryKey"`
	AMCAssignmentID uuid.UUID  `json:"amc_assignment_id" gorm:"type:uuid;index;not null"`
	EventType       string     `json:"event_type" gorm:"type:varchar(32);index;not null"`
	ActorUserID     uuid.UUID  `json:"actor_user_id" gorm:"type:uuid;index;not null"`
	FromEngineerID  *uuid.UUID `json:"from_engineer_id,omitempty" gorm:"type:uuid"`
	ToEngineerID    uuid.UUID  `json:"to_engineer_id" gorm:"type:uuid;index;not null"`
	Note            string     `json:"note,omitempty" gorm:"type:text"`
	CreatedAt       time.Time  `json:"created_at"`

	Actor        *User            `json:"actor,omitempty" gorm:"foreignKey:ActorUserID"`
	FromEngineer *SupportEngineer `json:"from_engineer,omitempty" gorm:"foreignKey:FromEngineerID"`
	ToEngineer   *SupportEngineer `json:"to_engineer,omitempty" gorm:"foreignKey:ToEngineerID"`
}

func (AMCAssignmentEvent) TableName() string {
	return "amc_assignment_events"
}
