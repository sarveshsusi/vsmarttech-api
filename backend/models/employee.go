package models

import (
	"time"

	"github.com/google/uuid"
)

/* =========================
   SUPPORT ENGINEER
========================= */

type SupportEngineer struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;uniqueIndex"` // RoleSupport
	Designation string    `json:"designation" gorm:"type:varchar(100)"`
	Phone       string    `json:"phone" gorm:"type:varchar(20)"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	User User `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (SupportEngineer) TableName() string {
	return "support_engineers"
}

/* =========================
   SERVICE VISIT (ticket field visits)
========================= */

type ServiceVisit struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID   string    `json:"ticket_id" gorm:"type:varchar(20);index;not null"`
	EngineerID uuid.UUID `json:"engineer_id" gorm:"type:uuid;index;not null"` // assigned engineer who logged the visit
	VisitDate  time.Time `json:"visit_date" gorm:"type:date;not null"`
	Notes      string    `json:"notes" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Keep legacy columns nullable so AutoMigrate does not break existing rows
	StartTime *time.Time `json:"-" gorm:"column:start_time"`
	EndTime   *time.Time `json:"-" gorm:"column:end_time"`

	Engineer    *SupportEngineer    `json:"logged_by,omitempty" gorm:"foreignKey:EngineerID"`
	CoEngineers []SupportEngineer   `json:"co_engineers,omitempty" gorm:"many2many:service_visit_co_engineers;"`
	Proofs      []ServiceVisitProof `json:"proofs,omitempty" gorm:"foreignKey:ServiceVisitID"`
	Ticket      *Ticket             `json:"ticket,omitempty" gorm:"foreignKey:TicketID;references:ID"`
}

func (ServiceVisit) TableName() string {
	return "service_visits"
}

/* =========================
   SERVICE VISIT PROOF
========================= */

type ServiceVisitProof struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ServiceVisitID uuid.UUID `json:"service_visit_id" gorm:"type:uuid;index;not null"`
	URL            string    `json:"url" gorm:"type:text;not null"`
	FileName       string    `json:"file_name" gorm:"type:varchar(255)"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ServiceVisitProof) TableName() string {
	return "service_visit_proofs"
}

/* =========================
   GPS LOG
========================= */

type GPSLog struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	EngineerID uuid.UUID `json:"engineer_id" gorm:"type:uuid"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	LoggedAt   time.Time `json:"logged_at"`
}

func (GPSLog) TableName() string {
	return "gps_logs"
}

/* =========================
   DIGITAL SIGNATURE
========================= */

type DigitalSignature struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID string    `json:"ticket_id" gorm:"type:varchar(20)"`
	SignedBy string    `json:"signed_by"`
	FileURL  string    `json:"file_url"`
	SignedAt time.Time `json:"signed_at"`
}

func (DigitalSignature) TableName() string {
	return "digital_signatures"
}
