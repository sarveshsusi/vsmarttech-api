package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
	"rbac/repository"
)

type CustomerSolutionService struct {
	db           *gorm.DB
	repo         *repository.CustomerSolutionRepository
	customerRepo *repository.CustomerRepository
}

func NewCustomerSolutionService(
	db *gorm.DB,
	repo *repository.CustomerSolutionRepository,
	customerRepo *repository.CustomerRepository,
) *CustomerSolutionService {
	return &CustomerSolutionService{
		db:           db,
		repo:         repo,
		customerRepo: customerRepo,
	}
}

/* =========================
   ADMIN: ASSIGN SOLUTION
========================= */

type AssignSolutionRequest struct {
	CustomerID   uuid.UUID
	SolutionID   uuid.UUID
	PONumber     string
	ContractType models.ContractType
	Description  string
	AMCType      *models.AMCType
	AMCStartDate *time.Time
	AMCEndDate   *time.Time

	WarrantyStartDate *time.Time
	WarrantyEndDate   *time.Time

	ChargeableType *models.ChargeableType

	AssignedBy uuid.UUID
}

func (s *CustomerSolutionService) AssignSolution(req *AssignSolutionRequest) error {

	if req.PONumber == "" {
		return errors.New("po number required")
	}

	// ✅ Verify customer exists - check both by customer ID and user ID
	exists, err := s.customerRepo.ExistsByID(req.CustomerID)
	if err != nil {
		return errors.New("failed to verify customer")
	}

	if !exists {
		// Try to find customer by user_id (in case user_id was passed instead)
		customer, err := s.customerRepo.GetByUserID(req.CustomerID)
		if err != nil || customer == nil {
			return errors.New("customer not found or inactive")
		}
		// Update the request to use the actual customer ID
		req.CustomerID = customer.ID
	}

	if req.ContractType == models.ContractAMC {
		if req.AMCType == nil || req.AMCStartDate == nil || req.AMCEndDate == nil {
			return errors.New("invalid AMC details")
		}
	}

	if req.ContractType == models.ContractWarranty {
		if req.WarrantyStartDate == nil || req.WarrantyEndDate == nil {
			return errors.New("invalid warranty details")
		}
	}

	if req.ContractType == models.ContractOthers {
		if req.ChargeableType == nil {
			return errors.New("invalid chargeable type")
		}
	}

	cs := &models.CustomerSolution{
		CustomerID:   req.CustomerID,
		SolutionID:   req.SolutionID,
		PONumber:     req.PONumber,
		ContractType: req.ContractType,

		Description: req.Description,

		AMCType:      req.AMCType,
		AMCStartDate: req.AMCStartDate,
		AMCEndDate:   req.AMCEndDate,

		WarrantyStartDate: req.WarrantyStartDate,
		WarrantyEndDate:   req.WarrantyEndDate,

		ChargeableType: req.ChargeableType,

		AssignedBy: req.AssignedBy,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}

	return s.repo.Create(cs)
}

/* =========================
   READ
========================= */

func (s *CustomerSolutionService) GetCustomerSolutions(customerID uuid.UUID) ([]models.CustomerSolution, error) {
	return s.repo.GetByCustomer(customerID)
}

func (s *CustomerSolutionService) GetCustomerSolutionsByUserID(
	userID uuid.UUID,
) ([]models.CustomerSolution, error) {

	customer, err := s.customerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("customer profile not found")
	}

	return s.repo.GetByCustomer(customer.ID)
}

func (s *CustomerSolutionService) GetAll() ([]models.CustomerSolution, error) {
	return s.repo.GetAll()
}
