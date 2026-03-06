// service/amc_service.go
package service

import (
	"errors"
	"time"

	"rbac/models"
	"rbac/repository"

	"github.com/google/uuid"
)

type AMCService struct {
	repo *repository.AMCRepository
}

func NewAMCService(r *repository.AMCRepository) *AMCService {
	return &AMCService{repo: r}
}

// Admin
func (s *AMCService) GetAllAMCs() ([]models.AMCContract, error) {
	return s.repo.FindAll()
}

// Customer
func (s *AMCService) GetCustomerAMCs(
	userID uuid.UUID,
	role models.Role,
) ([]models.AMCContract, error) {

	if role != models.RoleCustomer {
		return nil, errors.New("not a customer")
	}

	return s.repo.FindByCustomerUserID(userID)
}
func (s *AMCService) CreateAMC(
	customerProductID uuid.UUID,
	slaHours int,
	start time.Time,
	end time.Time,
) (*models.AMCContract, error) {

	amc := &models.AMCContract{
		CustomerProductID: customerProductID,
		SLAHours:          slaHours,
		StartDate:         start,
		EndDate:           end,
		Status:            "active",
	}

	return amc, s.repo.Create(amc)
}
