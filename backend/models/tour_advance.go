package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TourAdvanceStatus string

const (
	TourStatusPendingApproval TourAdvanceStatus = "PENDING_APPROVAL"
	TourStatusApproved        TourAdvanceStatus = "APPROVED"
	TourStatusRejected        TourAdvanceStatus = "REJECTED"
	TourStatusBillSubmitted   TourAdvanceStatus = "BILL_SUBMITTED"
	TourStatusProcessed       TourAdvanceStatus = "PROCESSED"
)

type TourWorkType string

const (
	TourWorkInstallation TourWorkType = "INSTALLATION"
	TourWorkService      TourWorkType = "SERVICE"
)

type TourTravelMode string

const (
	TourTravelBus   TourTravelMode = "BUS"
	TourTravelTrain TourTravelMode = "TRAIN"
	TourTravelCar   TourTravelMode = "CAR"
)

type TourAdvanceEventAction string

const (
	TourEventCreated       TourAdvanceEventAction = "created"
	TourEventApproved      TourAdvanceEventAction = "approved"
	TourEventRejected      TourAdvanceEventAction = "rejected"
	TourEventBillSubmitted TourAdvanceEventAction = "bill_submitted"
	TourEventProcessed     TourAdvanceEventAction = "processed"
)

type TourAdvanceRequest struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	RequestNumber string    `gorm:"type:varchar(16);uniqueIndex;not null" json:"request_number"`

	EngineerID uuid.UUID        `gorm:"type:uuid;not null;index" json:"engineer_id"`
	Engineer   *SupportEngineer `gorm:"foreignKey:EngineerID;constraint:OnDelete:RESTRICT" json:"-"`

	CustomerID       uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	Customer         *Customer `gorm:"foreignKey:CustomerID;constraint:OnDelete:RESTRICT" json:"-"`
	LocationSnapshot string    `gorm:"type:varchar(150);not null" json:"location_snapshot"`

	WorkType TourWorkType `gorm:"type:varchar(20);not null" json:"work_type"`
	PONumber *string      `gorm:"type:varchar(100)" json:"po_number,omitempty"`
	TicketID *string      `gorm:"type:varchar(20);index" json:"ticket_id,omitempty"`
	Ticket   *Ticket      `gorm:"foreignKey:TicketID;constraint:OnDelete:RESTRICT" json:"-"`

	PersonsTravelling int `gorm:"not null" json:"persons_travelling"`
	DaysPlanned       int `gorm:"not null" json:"days_planned"`

	FoodExpense          decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"food_expense"`
	LocalExpense         decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"local_expense"`
	AccommodationExpense decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"accommodation_expense"`
	TravelMode           TourTravelMode  `gorm:"type:varchar(10);not null" json:"travel_mode"`
	TravelExpense        decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"travel_expense"`

	RequestedAmount decimal.Decimal  `gorm:"type:numeric(12,2);not null" json:"requested_amount"`
	ApprovedAmount  *decimal.Decimal `gorm:"type:numeric(12,2)" json:"approved_amount,omitempty"`

	Status TourAdvanceStatus `gorm:"type:varchar(32);not null;index;index:idx_tour_advance_status_created,priority:1" json:"status"`

	ApprovalRemarks string     `gorm:"type:text" json:"approval_remarks"`
	ApprovedBy      *uuid.UUID `gorm:"type:uuid;index" json:"approved_by,omitempty"`
	ApprovedByUser  *User      `gorm:"foreignKey:ApprovedBy;references:ID;constraint:OnDelete:RESTRICT" json:"-"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`

	RejectionRemarks string     `gorm:"type:text" json:"rejection_remarks"`
	RejectedBy       *uuid.UUID `gorm:"type:uuid;index" json:"rejected_by,omitempty"`
	RejectedByUser   *User      `gorm:"foreignKey:RejectedBy;references:ID;constraint:OnDelete:RESTRICT" json:"-"`
	RejectedAt       *time.Time `json:"rejected_at,omitempty"`

	BillSubmittedBy     *uuid.UUID `gorm:"type:uuid;index" json:"bill_submitted_by,omitempty"`
	BillSubmittedByUser *User      `gorm:"foreignKey:BillSubmittedBy;references:ID;constraint:OnDelete:RESTRICT" json:"-"`
	BillSubmittedAt     *time.Time `json:"bill_submitted_at,omitempty"`

	ProcessedBy     *uuid.UUID `gorm:"type:uuid;index" json:"processed_by,omitempty"`
	ProcessedByUser *User      `gorm:"foreignKey:ProcessedBy;references:ID;constraint:OnDelete:RESTRICT" json:"-"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`

	Events []TourAdvanceEvent `gorm:"foreignKey:TourAdvanceRequestID" json:"-"`

	CreatedAt time.Time `gorm:"index:idx_tour_advance_status_created,priority:2" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TourAdvanceRequest) TableName() string {
	return "tour_advance_requests"
}

type TourAdvanceEvent struct {
	ID                   uuid.UUID              `gorm:"type:uuid;primaryKey" json:"id"`
	TourAdvanceRequestID uuid.UUID              `gorm:"type:uuid;not null;index" json:"tour_advance_request_id"`
	Action               TourAdvanceEventAction `gorm:"type:varchar(32);not null" json:"action"`
	ActorUserID          uuid.UUID              `gorm:"type:uuid;not null;index" json:"actor_user_id"`
	Actor                *User                  `gorm:"foreignKey:ActorUserID;constraint:OnDelete:RESTRICT" json:"-"`
	Note                 string                 `gorm:"type:text" json:"note"`
	CreatedAt            time.Time              `json:"created_at"`
}

func (TourAdvanceEvent) TableName() string {
	return "tour_advance_events"
}
