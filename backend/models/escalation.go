package models

import (
	"time"

	"github.com/google/uuid"
)

type EscalationRule struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Condition string    `gorm:"type:varchar(50)"` // unassigned / overdue
	AfterMins int
	Role      Role
	CreatedAt time.Time
}

func (EscalationRule) TableName() string {
	return "escalation_rules"
}

type TicketEscalation struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID    string    `gorm:"type:varchar(20)"`
	RuleID      uuid.UUID `gorm:"type:uuid"`
	EscalatedAt time.Time
	Resolved    bool
}

func (TicketEscalation) TableName() string {
	return "ticket_escalations"
}
