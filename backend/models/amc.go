package models

import (
	"time"

	"github.com/google/uuid"
)

// models/amc_contract.go

type AMCContract struct {
	ID                uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CustomerProductID uuid.UUID `gorm:"type:uuid;index"`
	SLAHours          int
	StartDate         time.Time
	EndDate           time.Time
	Status            string // active, expired
	CreatedAt         time.Time
}

func (AMCContract) TableName() string {
	return "amc_contracts"
}

type AMCSchedule struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AMCID     uuid.UUID `gorm:"type:uuid;index"`
	VisitDate time.Time
	Completed bool
	TicketID  *uuid.UUID `gorm:"type:uuid"`

	AMC AMCContract `gorm:"foreignKey:AMCID"`
}

func (AMCSchedule) TableName() string {
	return "amc_schedules"
}
