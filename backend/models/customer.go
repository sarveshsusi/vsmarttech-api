package models

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"user_id"`
	CompanyID uuid.UUID `gorm:"type:uuid;index" json:"company_id"`

	Name          string `gorm:"type:varchar(150);not null" json:"name"`
	Address       string `gorm:"type:text" json:"address"`
	Location      string `gorm:"type:varchar(100)" json:"location"`
	Plant         string `gorm:"type:varchar(100)" json:"plant"`
	Phone         string `gorm:"type:varchar(20)" json:"phone"`
	Email         string `gorm:"type:varchar(150)" json:"email"`
	ContactPerson string `gorm:"type:varchar(150)" json:"contact_person"`

	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User    User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Company Company `gorm:"foreignKey:CompanyID" json:"company"`
}

func (Customer) TableName() string {
	return "customers"
}
