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
   SERVICE VISIT
========================= */

type ServiceVisit struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID   string     `json:"ticket_id" gorm:"type:varchar(20)"`
	EngineerID uuid.UUID  `json:"engineer_id" gorm:"type:uuid"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Notes      string     `json:"notes"`
}

func (ServiceVisit) TableName() string {
	return "service_visits"
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
