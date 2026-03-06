package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SupportEngineerRepository struct {
	db *gorm.DB
}

func NewSupportEngineerRepository(db *gorm.DB) *SupportEngineerRepository {
	return &SupportEngineerRepository{db: db}
}

func (r *SupportEngineerRepository) Create(
	tx *gorm.DB,
	engineer *models.SupportEngineer,
) error {
	return tx.Create(engineer).Error
}

func (r *SupportEngineerRepository) GetAll() ([]models.SupportEngineer, error) {
	var engineers []models.SupportEngineer
	err := r.db.
		Preload("User").
		Order("created_at DESC").
		Find(&engineers).Error

	// Debug logging
	for _, eng := range engineers {
		println("Engineer:", eng.ID.String(), "User Name:", eng.User.Name, "User Email:", eng.User.Email)
	}

	return engineers, err
}

func (r *SupportEngineerRepository) GetAllActive() ([]models.SupportEngineer, error) {
	var engineers []models.SupportEngineer
	err := r.db.
		Preload("User").
		Where("is_active = true").
		Find(&engineers).Error
	return engineers, err
}

func (r *SupportEngineerRepository) GetByUserID(userID uuid.UUID) (*models.SupportEngineer, error) {
	var engineer models.SupportEngineer
	err := r.db.Where("user_id = ? AND is_active = true", userID).
		First(&engineer).Error
	return &engineer, err
}

func (r *SupportEngineerRepository) GetByID(id uuid.UUID) (*models.SupportEngineer, error) {
	var engineer models.SupportEngineer
	err := r.db.Where("id = ?", id).
		Preload("User").
		First(&engineer).Error
	return &engineer, err
}
