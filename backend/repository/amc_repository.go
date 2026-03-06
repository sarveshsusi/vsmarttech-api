// repository/amc_repository.go
package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
)

type AMCRepository struct {
	db *gorm.DB
}

func NewAMCRepository(db *gorm.DB) *AMCRepository {
	return &AMCRepository{db: db}
}

// Admin: fetch all AMC contracts
func (r *AMCRepository) FindAll() ([]models.AMCContract, error) {
	var amcs []models.AMCContract

	err := r.db.
		Order("end_date ASC").
		Find(&amcs).
		Error

	return amcs, err
}

// Customer: fetch AMC contracts using CUSTOMER'S USER ID (UUID)
func (r *AMCRepository) FindByCustomerUserID(
	userID uuid.UUID,
) ([]models.AMCContract, error) {

	var amcs []models.AMCContract

	err := r.db.
		Joins("JOIN customers c ON c.id = amc_contracts.customer_id").
		Where("c.user_id = ?", userID).
		Order("amc_contracts.end_date ASC").
		Find(&amcs).
		Error

	return amcs, err
}
func (r *AMCRepository) Create(contract *models.AMCContract) error {
	return r.db.Create(contract).Error
}
