package repository

import (
	"errors"
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(
	tx *gorm.DB,
	customer *models.Customer,
) error {
	return tx.Create(customer).Error
}

func (r *CustomerRepository) FindByUserID(
	userID uuid.UUID,
) (*models.Customer, error) {

	var customer models.Customer

	err := r.db.
		Preload("Company").
		Where("user_id = ?", userID).
		First(&customer).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	return &customer, nil
}

func (r *CustomerRepository) GetAllPaginated(
	page int,
	limit int,
) ([]models.Customer, int64, error) {

	var customers []models.Customer
	var total int64

	offset := (page - 1) * limit

	r.db.Model(&models.Customer{}).Count(&total)

	err := r.db.
		Preload("Company").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&customers).Error

	return customers, total, err
}
func (r *CustomerRepository) GetByUserID(
	userID uuid.UUID,
) (*models.Customer, error) {

	var customer models.Customer

	err := r.db.
		Preload("Company").
		Where("user_id = ?", userID).
		First(&customer).
		Error

	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (r *CustomerRepository) ExistsByID(
	customerID uuid.UUID,
) (bool, error) {

	var count int64
	err := r.db.
		Model(&models.Customer{}).
		Where("(id = ? OR user_id = ?) AND is_active = true", customerID, customerID).
		Count(&count).Error

	return count > 0, err
}

// GetByID returns a customer by their ID with user and company info
func (r *CustomerRepository) GetByID(
	customerID uuid.UUID,
) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.
		Preload("User").
		Preload("Company").
		Where("id = ? AND is_active = true", customerID).
		First(&customer).Error

	if err != nil {
		return nil, err
	}
	return &customer, nil
}
