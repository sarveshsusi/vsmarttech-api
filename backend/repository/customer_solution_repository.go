package repository

import (
	"errors"
	"time"

	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerSolutionRepository struct {
	db *gorm.DB
}

func NewCustomerSolutionRepository(db *gorm.DB) *CustomerSolutionRepository {
	return &CustomerSolutionRepository{db: db}
}

func (r *CustomerSolutionRepository) Create(cs *models.CustomerSolution) error {
	return r.db.Create(cs).Error
}

func (r *CustomerSolutionRepository) GetByPO(
	poNumber string,
) (*models.CustomerSolution, error) {

	var cs models.CustomerSolution
	err := r.db.
		Preload("Solution").
		Where("po_number = ?", poNumber).
		First(&cs).Error

	if err != nil {
		return nil, errors.New("contract not found")
	}

	return &cs, nil
}

func (r *CustomerSolutionRepository) IsPOExpired(
	cs *models.CustomerSolution,
) bool {

	now := time.Now()

	if cs.ContractType == models.ContractAMC && cs.AMCEndDate != nil {
		return cs.AMCEndDate.Before(now)
	}

	if cs.ContractType == models.ContractWarranty && cs.WarrantyEndDate != nil {
		return cs.WarrantyEndDate.Before(now)
	}

	return false
}

func (r *CustomerSolutionRepository) GetByCustomerAndPO(
	customerID uuid.UUID,
	po string,
) (*models.CustomerSolution, error) {

	var cs models.CustomerSolution
	err := r.db.
		Preload("Solution").
		Where("customer_id = ? AND po_number = ? AND is_active = true", customerID, po).
		First(&cs).Error

	return &cs, err
}

func (r *CustomerSolutionRepository) GetByID(
	id uuid.UUID,
) (*models.CustomerSolution, error) {

	var cs models.CustomerSolution

	err := r.db.
		Preload("Solution").
		First(&cs, "id = ? AND is_active = true", id).
		Error

	return &cs, err
}
func (r *CustomerSolutionRepository) GetByCustomer(
	customerID uuid.UUID,
) ([]models.CustomerSolution, error) {

	var cs []models.CustomerSolution
	err := r.db.
		Preload("Solution").
		Where("customer_id = ?", customerID).
		Find(&cs).Error

	return cs, err
}

// GetExpiringAMCs returns AMC contracts expiring within the given days
func (r *CustomerSolutionRepository) GetExpiringAMCs(daysUntilExpiry int) ([]models.CustomerSolution, error) {
	var contracts []models.CustomerSolution

	now := time.Now().Truncate(24 * time.Hour)
	targetDate := now.AddDate(0, 0, daysUntilExpiry)

	// Find contracts expiring between now and targetDate (inclusive)
	err := r.db.
		Preload("Solution").
		Preload("Customer").
		Preload("Customer.User").
		Where("contract_type = ? AND is_active = true AND amc_end_date >= ? AND amc_end_date <= ?",
			models.ContractAMC, now, targetDate).
		Find(&contracts).Error

	return contracts, err
}

// GetExpiringWarranties returns Warranty contracts expiring within the given days
func (r *CustomerSolutionRepository) GetExpiringWarranties(daysUntilExpiry int) ([]models.CustomerSolution, error) {
	var contracts []models.CustomerSolution

	now := time.Now().Truncate(24 * time.Hour)
	targetDate := now.AddDate(0, 0, daysUntilExpiry)

	// Find contracts expiring between now and targetDate (inclusive)
	err := r.db.
		Preload("Solution").
		Preload("Customer").
		Preload("Customer.User").
		Where("contract_type = ? AND is_active = true AND warranty_end_date >= ? AND warranty_end_date <= ?",
			models.ContractWarranty, now, targetDate).
		Find(&contracts).Error

	return contracts, err
}

// GetAllAMCContracts returns all active AMC contracts with customer info
func (r *CustomerSolutionRepository) GetAllAMCContracts() ([]models.CustomerSolution, error) {
	var contracts []models.CustomerSolution

	err := r.db.
		Preload("Solution").
		Where("contract_type = ? AND is_active = true", models.ContractAMC).
		Order("amc_end_date ASC").
		Find(&contracts).Error

	return contracts, err
}

// GetAllWarrantyContracts returns all active Warranty contracts
func (r *CustomerSolutionRepository) GetAllWarrantyContracts() ([]models.CustomerSolution, error) {
	var contracts []models.CustomerSolution

	err := r.db.
		Preload("Solution").
		Where("contract_type = ? AND is_active = true", models.ContractWarranty).
		Order("warranty_end_date ASC").
		Find(&contracts).Error

	return contracts, err
}

// GetAll returns all customer solutions (POs)
func (r *CustomerSolutionRepository) GetAll() ([]models.CustomerSolution, error) {
	var solutions []models.CustomerSolution

	err := r.db.
		Preload("Customer").
		Preload("Solution").
		Where("is_active = true").
		Order("created_at DESC").
		Find(&solutions).Error

	return solutions, err
}
