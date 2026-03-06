package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SolutionRepository struct {
	db *gorm.DB
}

func NewSolutionRepository(db *gorm.DB) *SolutionRepository {
	return &SolutionRepository{db: db}
}

func (r *SolutionRepository) Create(s *models.Solution) error {
	return r.db.Create(s).Error
}

func (r *SolutionRepository) GetAll() ([]models.Solution, error) {
	var list []models.Solution
	err := r.db.Where("is_active = true").Find(&list).Error
	return list, err
}

func (r *SolutionRepository) GetByID(id uuid.UUID) (*models.Solution, error) {
	var solution models.Solution
	err := r.db.First(&solution, "id = ?", id).Error
	return &solution, err
}

func (r *SolutionRepository) Update(s *models.Solution) error {
	return r.db.Model(s).Updates(s).Error
}

func (r *SolutionRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.Solution{}).Error
}
