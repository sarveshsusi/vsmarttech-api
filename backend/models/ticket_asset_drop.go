package models

import (
	"time"

	"github.com/google/uuid"
)

type AssetDropStatus string

const (
	AssetDropStatusRequested      AssetDropStatus = "requested"
	AssetDropStatusAcknowledged   AssetDropStatus = "acknowledged"
	AssetDropStatusReturnAssigned AssetDropStatus = "return_assigned"
	AssetDropStatusReturned       AssetDropStatus = "returned"
)

// TicketAssetDrop tracks a support-initiated device intake from a customer site
// through workshop acknowledgement, return assignment, and send-to-site.
type TicketAssetDrop struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	TicketID string  `gorm:"type:varchar(20);index;not null" json:"ticket_id"`
	Ticket   *Ticket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`

	SerialNumber string `gorm:"type:varchar(120);not null" json:"serial_number"`
	Name         string `gorm:"type:varchar(200);not null" json:"name"`
	Model        string `gorm:"type:varchar(120)" json:"model"`
	Category     string `gorm:"type:varchar(80)" json:"category"`
	SiteLocation string `gorm:"type:varchar(255)" json:"site_location"`
	IsReplacement bool  `gorm:"not null;default:false" json:"is_replacement"`

	Status AssetDropStatus `gorm:"type:varchar(32);index;not null;default:'requested'" json:"status"`

	AssetID *uuid.UUID `gorm:"type:uuid;index" json:"asset_id,omitempty"`
	Asset   *Asset     `gorm:"foreignKey:AssetID" json:"asset,omitempty"`

	ReturnEngineerID *uuid.UUID       `gorm:"type:uuid;index" json:"return_engineer_id,omitempty"`
	ReturnEngineer   *SupportEngineer `gorm:"foreignKey:ReturnEngineerID" json:"return_engineer,omitempty"`

	RequestedBy     uuid.UUID  `gorm:"type:uuid;not null" json:"requested_by"`
	AcknowledgedBy  *uuid.UUID `gorm:"type:uuid" json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`
	ReturnAssignedAt *time.Time `json:"return_assigned_at,omitempty"`
	ReturnedAt      *time.Time `json:"returned_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TicketAssetDrop) TableName() string {
	return "ticket_asset_drops"
}

// IsActive reports whether the drop still blocks a new request on the same ticket.
func (d AssetDropStatus) IsActive() bool {
	switch d {
	case AssetDropStatusRequested, AssetDropStatusAcknowledged, AssetDropStatusReturnAssigned:
		return true
	default:
		return false
	}
}
