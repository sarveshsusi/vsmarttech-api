package models

import (
	"time"

	"github.com/google/uuid"
)

type ContractType string
type AMCType string
type ChargeableType string

const (
	ContractAMC      ContractType = "AMC"
	ContractWarranty ContractType = "Warranty"
	ContractOthers   ContractType = "Others/Chargeable"

	AMCComprehensive    AMCType = "Comprehensive"
	AMCNonComprehensive AMCType = "Non-Comprehensive"

	ChargeableChargeable ChargeableType = "Chargeable"
	ChargeableOthers     ChargeableType = "Others"
)

type CustomerSolution struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CustomerID uuid.UUID `gorm:"type:uuid;index;not null" json:"customer_id"`
	SolutionID uuid.UUID `gorm:"type:uuid;index;not null" json:"solution_id"`

	PONumber     string       `gorm:"type:varchar(100);unique;not null" json:"po_number"`
	ContractType ContractType `gorm:"type:varchar(20);not null" json:"contract_type"`

	Description string `gorm:"type:text" json:"description"`

	// AMC
	AMCType      *AMCType   `json:"amc_type,omitempty"`
	AMCStartDate *time.Time `json:"amc_start_date,omitempty"`
	AMCEndDate   *time.Time `json:"amc_end_date,omitempty"`

	// Warranty
	WarrantyStartDate *time.Time `json:"warranty_start_date,omitempty"`
	WarrantyEndDate   *time.Time `json:"warranty_end_date,omitempty"`

	// Chargeable/Others
	ChargeableType *ChargeableType `json:"chargeable_type,omitempty"`

	IsActive   bool      `gorm:"default:true" json:"is_active"`
	AssignedBy uuid.UUID `gorm:"type:uuid;not null" json:"assigned_by"`
	CreatedAt  time.Time `json:"created_at"`

	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer"`
	Solution Solution `gorm:"foreignKey:SolutionID" json:"solution"`
}

func (CustomerSolution) TableName() string {
	return "customer_solutions"
}

// AMCCustomerNameHistory records display-name changes made from AMC contract edit.
type AMCCustomerNameHistory struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CustomerSolutionID uuid.UUID `json:"customer_solution_id" gorm:"type:uuid;index;not null"`
	CustomerID         uuid.UUID `json:"customer_id" gorm:"type:uuid;index;not null"`
	PONumber           string    `json:"po_number" gorm:"type:varchar(100);index;not null"`
	OldName            string    `json:"old_name" gorm:"type:varchar(150);not null"`
	NewName            string    `json:"new_name" gorm:"type:varchar(150);not null"`
	ChangedBy          uuid.UUID `json:"changed_by" gorm:"type:uuid;index;not null"`
	ChangedAt          time.Time `json:"changed_at" gorm:"index"`
	Note               string    `json:"note,omitempty" gorm:"type:varchar(255)"`

	ChangedByUser *User `json:"changed_by_user,omitempty" gorm:"foreignKey:ChangedBy"`
}

func (AMCCustomerNameHistory) TableName() string {
	return "amc_customer_name_histories"
}
