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
	TourEventCreated             TourAdvanceEventAction = "created"
	TourEventApproved            TourAdvanceEventAction = "approved"
	TourEventRejected            TourAdvanceEventAction = "rejected"
	TourEventBillSubmitted       TourAdvanceEventAction = "bill_submitted"
	TourEventProcessed           TourAdvanceEventAction = "processed"
	TourEventAdditionalRequested TourAdvanceEventAction = "additional_requested"
	TourEventAdditionalApproved  TourAdvanceEventAction = "additional_approved"
	TourEventAdditionalRejected  TourAdvanceEventAction = "additional_rejected"
	TourEventDisbursed           TourAdvanceEventAction = "disbursed"
	TourEventExpenseVerified     TourAdvanceEventAction = "expense_verified"
	TourEventExpenseRejected     TourAdvanceEventAction = "expense_rejected"
	TourEventExpenseClarified    TourAdvanceEventAction = "expense_clarified"
	TourEventExpenseAmended      TourAdvanceEventAction = "expense_amended"
	TourEventCreditIssued        TourAdvanceEventAction = "credit_issued"
	TourEventCreditApplied       TourAdvanceEventAction = "credit_applied"
	TourEventReady               TourAdvanceEventAction = "ready_for_processing"
	TourEventBlocked             TourAdvanceEventAction = "request_blocked"
	TourEventFlagged             TourAdvanceEventAction = "flagged"
)

type TourAdditionalStatus string

const (
	TourAdditionalPending  TourAdditionalStatus = "PENDING_APPROVAL"
	TourAdditionalApproved TourAdditionalStatus = "APPROVED"
	TourAdditionalRejected TourAdditionalStatus = "REJECTED"
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

	PersonsTravelling int     `gorm:"not null" json:"persons_travelling"`
	Travellers        string  `gorm:"type:jsonb" json:"-"`
	DaysPlanned       int     `gorm:"not null" json:"days_planned"`
	TravelFrom        *string `gorm:"type:varchar(10)" json:"travel_from,omitempty"`
	TravelTo          *string `gorm:"type:varchar(10)" json:"travel_to,omitempty"`

	FoodExpense          decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"food_expense"`
	LocalExpense         decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"local_expense"`
	AccommodationExpense decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"accommodation_expense"`
	TravelMode           TourTravelMode  `gorm:"type:varchar(10);not null" json:"travel_mode"`
	TravelExpense        decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"travel_expense"`

	RequestedAmount       decimal.Decimal  `gorm:"type:numeric(12,2);not null" json:"requested_amount"`
	ApprovedAmount        *decimal.Decimal `gorm:"type:numeric(12,2)" json:"approved_amount,omitempty"`
	CreditApplied         decimal.Decimal  `gorm:"type:numeric(12,2);not null;default:0" json:"credit_applied"`
	NetPayable            *decimal.Decimal `gorm:"type:numeric(12,2)" json:"net_payable,omitempty"`
	ApprovalJustification string           `gorm:"type:text" json:"approval_justification"`
	PaymentMode           *string          `gorm:"type:varchar(16)" json:"payment_mode,omitempty"`
	TransactionID         *string          `gorm:"type:varchar(64)" json:"transaction_id,omitempty"`

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

	Events        []TourAdvanceEvent        `gorm:"foreignKey:TourAdvanceRequestID" json:"-"`
	Additional    *TourAdvanceAdditional    `gorm:"foreignKey:TourAdvanceRequestID" json:"-"`
	Expenses      []TourAdvanceExpense      `gorm:"foreignKey:TourAdvanceRequestID" json:"-"`
	Disbursements []TourAdvanceDisbursement `gorm:"foreignKey:TourAdvanceRequestID" json:"-"`
	Flags         []TourAdvanceFlag         `gorm:"foreignKey:TourAdvanceRequestID" json:"-"`

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

type TourAdvanceAdditional struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TourAdvanceRequestID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"tour_advance_request_id"`

	Amount decimal.Decimal      `gorm:"type:numeric(12,2);not null" json:"amount"`
	Reason string               `gorm:"type:text;not null" json:"reason"`
	Status TourAdditionalStatus `gorm:"type:varchar(32);not null;index" json:"status"`

	ApprovedAmount  *decimal.Decimal `gorm:"type:numeric(12,2)" json:"approved_amount,omitempty"`
	PaymentMode     *string          `gorm:"type:varchar(10)" json:"payment_mode,omitempty"`
	TransactionID   *string          `gorm:"type:varchar(64)" json:"transaction_id,omitempty"`
	ApprovalRemarks string           `gorm:"type:text" json:"approval_remarks"`
	ApprovedBy      *uuid.UUID       `gorm:"type:uuid;index" json:"approved_by,omitempty"`
	ApprovedByUser  *User            `gorm:"foreignKey:ApprovedBy;references:ID;constraint:OnDelete:RESTRICT" json:"-"`
	ApprovedAt      *time.Time       `json:"approved_at,omitempty"`

	RejectionRemarks string     `gorm:"type:text" json:"rejection_remarks"`
	RejectedBy       *uuid.UUID `gorm:"type:uuid;index" json:"rejected_by,omitempty"`
	RejectedByUser   *User      `gorm:"foreignKey:RejectedBy;references:ID;constraint:OnDelete:RESTRICT" json:"-"`
	RejectedAt       *time.Time `json:"rejected_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TourAdvanceAdditional) TableName() string {
	return "tour_advance_additionals"
}

type TourExpenseCategory string

const (
	TourExpenseFood          TourExpenseCategory = "FOOD"
	TourExpenseLocalTravel   TourExpenseCategory = "LOCAL_TRAVEL"
	TourExpenseAccommodation TourExpenseCategory = "ACCOMMODATION"
	TourExpenseTravel        TourExpenseCategory = "TRAVEL"
	TourExpenseOther         TourExpenseCategory = "OTHER"
)

type TourBillType string

const (
	TourBillTypeBill    TourBillType = "BILL"
	TourBillTypeNonBill TourBillType = "NON_BILL"
)

type TourExpenseStatus string

const (
	TourExpensePendingReview      TourExpenseStatus = "PENDING_REVIEW"
	TourExpenseNeedsClarification TourExpenseStatus = "NEEDS_CLARIFICATION"
	TourExpenseVerified           TourExpenseStatus = "VERIFIED"
	TourExpenseRejected           TourExpenseStatus = "REJECTED"
)

type TourAdvanceExpense struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TourAdvanceRequestID uuid.UUID `gorm:"type:uuid;not null;index" json:"tour_advance_request_id"`
	EngineerID           uuid.UUID `gorm:"type:uuid;not null;index" json:"engineer_id"`

	Category        TourExpenseCategory `gorm:"type:varchar(20);not null" json:"category"`
	BillType        TourBillType        `gorm:"type:varchar(10);not null" json:"bill_type"`
	ExpenseDate     string              `gorm:"type:varchar(10);not null" json:"expense_date"`
	Description     string              `gorm:"type:varchar(500);not null" json:"description"`
	ClaimedAmount   decimal.Decimal     `gorm:"type:numeric(12,2);not null" json:"claimed_amount"`
	ReferenceNumber string              `gorm:"type:varchar(100);not null;default:''" json:"reference_number"`
	MissingReason   string              `gorm:"type:varchar(500);not null;default:''" json:"missing_reason"`

	AcceptedAmount *decimal.Decimal  `gorm:"type:numeric(12,2)" json:"accepted_amount,omitempty"`
	Status         TourExpenseStatus `gorm:"type:varchar(32);not null;index" json:"status"`
	ReviewReason   string            `gorm:"type:text;not null;default:''" json:"review_reason"`
	ReviewedBy     *uuid.UUID        `gorm:"type:uuid;index" json:"reviewed_by,omitempty"`
	ReviewedByUser *User             `gorm:"foreignKey:ReviewedBy;references:ID;constraint:OnDelete:RESTRICT" json:"-"`
	ReviewedAt     *time.Time        `json:"reviewed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TourAdvanceExpense) TableName() string {
	return "tour_advance_expenses"
}

type TourAdvanceDisbursement struct {
	ID                   uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	TourAdvanceRequestID uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_tour_disburse_source,priority:1" json:"tour_advance_request_id"`
	Source               string          `gorm:"type:varchar(16);not null;uniqueIndex:idx_tour_disburse_source,priority:2" json:"source"`
	Amount               decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"amount"`
	PaymentMode          string          `gorm:"type:varchar(16);not null" json:"payment_mode"`
	TransactionID        *string         `gorm:"type:varchar(64)" json:"transaction_id,omitempty"`
	ActorUserID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"actor_user_id"`
	IdempotencyKey       string          `gorm:"type:varchar(80);not null;uniqueIndex" json:"-"`
	CreatedAt            time.Time       `json:"created_at"`
}

func (TourAdvanceDisbursement) TableName() string {
	return "tour_advance_disbursements"
}

type TourAdvanceLedgerEntry struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	EngineerID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"engineer_id"`
	OriginatingTourID uuid.UUID       `gorm:"type:uuid;not null;index" json:"originating_tour_id"`
	ConsumingTourID   *uuid.UUID      `gorm:"type:uuid;index" json:"consuming_tour_id,omitempty"`
	EntryType         string          `gorm:"type:varchar(32);not null" json:"entry_type"`
	Amount            decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"amount"`
	BalanceBefore     decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"balance_before"`
	BalanceAfter      decimal.Decimal `gorm:"type:numeric(12,2);not null" json:"balance_after"`
	ActorUserID       uuid.UUID       `gorm:"type:uuid;not null" json:"actor_user_id"`
	Reason            string          `gorm:"type:text;not null;default:''" json:"reason"`
	IdempotencyKey    string          `gorm:"type:varchar(120);not null;uniqueIndex" json:"-"`
	CreatedAt         time.Time       `json:"created_at"`
}

func (TourAdvanceLedgerEntry) TableName() string {
	return "tour_advance_ledger"
}

type TourAdvanceFlag struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	TourAdvanceRequestID uuid.UUID `gorm:"type:uuid;not null;index" json:"tour_advance_request_id"`
	Kind                 string    `gorm:"type:varchar(32);not null" json:"kind"`
	Reason               string    `gorm:"type:varchar(300);not null" json:"reason"`
	CreatedAt            time.Time `json:"created_at"`
}

func (TourAdvanceFlag) TableName() string {
	return "tour_advance_flags"
}

type TourAdvancePolicy struct {
	ID                 uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	HighValueThreshold *decimal.Decimal `gorm:"type:numeric(12,2)" json:"high_value_threshold,omitempty"`
	UpdatedBy          *uuid.UUID       `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt          time.Time        `json:"updated_at"`
	Singleton          bool             `gorm:"not null;uniqueIndex" json:"-"`
}

func (TourAdvancePolicy) TableName() string {
	return "tour_advance_policies"
}
