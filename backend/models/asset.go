package models

import (
	"time"

	"github.com/google/uuid"
)

type AssetStatus string

const (
	AssetStatusAtSite         AssetStatus = "At Site"
	AssetStatusCollected      AssetStatus = "Collected"
	AssetStatusInWorkshop     AssetStatus = "In Workshop"
	AssetStatusReadyToReturn  AssetStatus = "Ready to Return"
	AssetStatusReturnedToSite AssetStatus = "Returned to Site"
	AssetStatusDecommissioned AssetStatus = "Decommissioned"

	// Legacy values still accepted when reading older rows.
	AssetStatusActive   AssetStatus = "Active"
	AssetStatusInactive AssetStatus = "Inactive"
)

// ValidAssetStatuses are material/service statuses used in the workshop flow.
var ValidAssetStatuses = []AssetStatus{
	AssetStatusAtSite,
	AssetStatusCollected,
	AssetStatusInWorkshop,
	AssetStatusReadyToReturn,
	AssetStatusReturnedToSite,
	AssetStatusDecommissioned,
}

// NormalizeAssetStatus maps legacy statuses and blanks to the current set.
func NormalizeAssetStatus(status AssetStatus) AssetStatus {
	switch status {
	case AssetStatusActive, AssetStatusInactive, "":
		return AssetStatusAtSite
	case AssetStatusAtSite,
		AssetStatusCollected,
		AssetStatusInWorkshop,
		AssetStatusReadyToReturn,
		AssetStatusReturnedToSite,
		AssetStatusDecommissioned:
		return status
	default:
		return AssetStatusAtSite
	}
}

// IsValidAssetStatus reports whether status is an allowed write value
// (including legacy Active/Inactive which normalize to At Site).
func IsValidAssetStatus(status AssetStatus) bool {
	switch status {
	case AssetStatusAtSite,
		AssetStatusCollected,
		AssetStatusInWorkshop,
		AssetStatusReadyToReturn,
		AssetStatusReturnedToSite,
		AssetStatusDecommissioned,
		AssetStatusActive,
		AssetStatusInactive,
		"":
		return true
	default:
		return false
	}
}

// Asset is an installed device (camera, barrier, biometric, etc.)
// linked to a company and optionally a PO / customer solution.
type Asset struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	CompanyID uuid.UUID `gorm:"type:uuid;index;not null" json:"company_id"`
	Company   Company   `gorm:"foreignKey:CompanyID" json:"company,omitempty"`

	CustomerID *uuid.UUID `gorm:"type:uuid;index" json:"customer_id,omitempty"`
	Customer   *Customer  `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`

	CustomerSolutionID *uuid.UUID        `gorm:"type:uuid;index" json:"customer_solution_id,omitempty"`
	CustomerSolution   *CustomerSolution `gorm:"foreignKey:CustomerSolutionID" json:"customer_solution,omitempty"`

	SerialNumber string `gorm:"type:varchar(120);uniqueIndex;not null" json:"serial_number"`
	Name         string `gorm:"type:varchar(200);not null" json:"name"`
	Model        string `gorm:"type:varchar(120)" json:"model"`
	Category     string `gorm:"type:varchar(80)" json:"category"` // Camera, Barrier, Biometric, Other
	SiteLocation string `gorm:"type:varchar(255)" json:"site_location"`
	Notes        string `gorm:"type:text" json:"notes"`

	Status AssetStatus `gorm:"type:varchar(32);default:'At Site'" json:"status"`

	InstalledAt *time.Time `json:"installed_at,omitempty"`
	CreatedBy   uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Asset) TableName() string {
	return "assets"
}
