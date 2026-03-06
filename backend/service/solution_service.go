package service

import (
	"rbac/models"
	"rbac/repository"

	"github.com/google/uuid"
)

type SolutionService struct {
	repo *repository.SolutionRepository
}

func NewSolutionService(r *repository.SolutionRepository) *SolutionService {
	return &SolutionService{repo: r}
}

func (s *SolutionService) Create(
	title string,
	description string,
	adminID uuid.UUID,
) (*models.Solution, error) {

	solution := &models.Solution{
		Title:       title,
		Description: description,
	}

	return solution, s.repo.Create(solution)
}

func (s *SolutionService) GetAll() ([]models.Solution, error) {
	return s.repo.GetAll()
}

func (s *SolutionService) Update(
	id uuid.UUID,
	title string,
	description string,
) (*models.Solution, error) {

	solution := &models.Solution{
		ID:          id,
		Title:       title,
		Description: description,
	}

	return solution, s.repo.Update(solution)
}

func (s *SolutionService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
