package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CompanyRepository interface {
	Create(company *models.Company) error
	FindAll() ([]models.Company, error)
	FindByID(id string) (*models.Company, error)
	Update(company *models.Company) error
	Delete(id uuid.UUID) error
}

type companyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) CompanyRepository {
	return &companyRepository{db}
}

func (r *companyRepository) Create(company *models.Company) error {
	return r.db.Create(company).Error
}

func (r *companyRepository) FindAll() ([]models.Company, error) {
	var companies []models.Company
	err := r.db.Order("name asc").Find(&companies).Error
	return companies, err
}

func (r *companyRepository) FindByID(id string) (*models.Company, error) {
	var company models.Company
	err := r.db.First(&company, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *companyRepository) Update(company *models.Company) error {
	return r.db.Model(company).Updates(company).Error
}

func (r *companyRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.Company{}).Error
}
