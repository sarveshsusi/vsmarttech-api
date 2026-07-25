package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
	"rbac/repository"
	"rbac/utils"
)

const maxCustomerDisplayNameLen = 150

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

/* =========================
   ADMIN: UPDATE CONTRACT (PO)
========================= */

type UpdateCustomerSolutionRequest struct {
	Description  *string
	ContractType *models.ContractType

	AMCType      *models.AMCType
	AMCStartDate *time.Time
	AMCEndDate   *time.Time

	WarrantyStartDate *time.Time
	WarrantyEndDate   *time.Time

	ChargeableType *models.ChargeableType

	IsActive *bool

	// CustomerName updates the linked login user's display name (same customer).
	CustomerName *string
	ChangedBy    uuid.UUID
}

func sanitizeCustomerDisplayName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	// Strip simple HTML tags to reduce stored XSS risk.
	for {
		start := strings.Index(s, "<")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], ">")
		if end < 0 {
			s = s[:start]
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("customer name is required")
	}
	if utf8.RuneCountInString(s) > maxCustomerDisplayNameLen {
		return "", errors.New("customer name is too long")
	}
	return s, nil
}

func (s *CustomerSolutionService) UpdateCustomerSolution(
	id uuid.UUID,
	req *UpdateCustomerSolutionRequest,
) error {
	cs, err := s.repo.GetByIDAny(id)
	if err != nil {
		return err
	}

	if req.ContractType != nil {
		switch *req.ContractType {
		case models.ContractAMC, models.ContractWarranty, models.ContractOthers:
		default:
			return errors.New("invalid contract type")
		}
	}

	// Resolve the effective value of every contract-type-specific field
	// (either the incoming update or the existing value) so we can validate
	// the whole record consistently, regardless of which fields were sent.
	contractType := cs.ContractType
	amcType := cs.AMCType
	amcStart := cs.AMCStartDate
	amcEnd := cs.AMCEndDate
	warrantyStart := cs.WarrantyStartDate
	warrantyEnd := cs.WarrantyEndDate
	chargeableType := cs.ChargeableType

	updates := map[string]interface{}{}

	if req.ContractType != nil {
		contractType = *req.ContractType
		updates["contract_type"] = *req.ContractType
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.AMCType != nil {
		amcType = req.AMCType
		updates["amc_type"] = req.AMCType
	}
	if req.AMCStartDate != nil {
		amcStart = req.AMCStartDate
		updates["amc_start_date"] = req.AMCStartDate
	}
	if req.AMCEndDate != nil {
		amcEnd = req.AMCEndDate
		updates["amc_end_date"] = req.AMCEndDate
	}
	if req.WarrantyStartDate != nil {
		warrantyStart = req.WarrantyStartDate
		updates["warranty_start_date"] = req.WarrantyStartDate
	}
	if req.WarrantyEndDate != nil {
		warrantyEnd = req.WarrantyEndDate
		updates["warranty_end_date"] = req.WarrantyEndDate
	}
	if req.ChargeableType != nil {
		chargeableType = req.ChargeableType
		updates["chargeable_type"] = req.ChargeableType
	}

	if contractType == models.ContractAMC {
		if amcType == nil || amcStart == nil || amcEnd == nil {
			return errors.New("invalid AMC details")
		}
		if amcStart.After(*amcEnd) {
			return errors.New("AMC start date must be before end date")
		}
	}

	if contractType == models.ContractWarranty {
		if warrantyStart == nil || warrantyEnd == nil {
			return errors.New("invalid warranty details")
		}
		if warrantyStart.After(*warrantyEnd) {
			return errors.New("warranty start date must be before end date")
		}
	}

	if contractType == models.ContractOthers {
		if chargeableType == nil {
			return errors.New("invalid chargeable type")
		}
	}

	var newName string
	var rename bool
	if req.CustomerName != nil {
		newName, err = sanitizeCustomerDisplayName(*req.CustomerName)
		if err != nil {
			return err
		}
		rename = true
	}

	if len(updates) == 0 && !rename {
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&models.CustomerSolution{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if !rename {
			return nil
		}

		var customer models.Customer
		if err := tx.Where("id = ?", cs.CustomerID).First(&customer).Error; err != nil {
			return errors.New("customer not found")
		}

		var user models.User
		if err := tx.Where("id = ?", customer.UserID).First(&user).Error; err != nil {
			return errors.New("customer user not found")
		}

		oldName := strings.TrimSpace(user.Name)
		if oldName == newName {
			return nil
		}

		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("name", newName).Error; err != nil {
			return err
		}

		// Keep contact_person aligned when it matched the previous display name, or was empty.
		if strings.TrimSpace(customer.ContactPerson) == "" ||
			strings.EqualFold(strings.TrimSpace(customer.ContactPerson), oldName) {
			_ = tx.Model(&models.Customer{}).
				Where("id = ?", customer.ID).
				Update("contact_person", newName).Error
		}

		changedBy := req.ChangedBy
		if changedBy == uuid.Nil {
			return errors.New("missing actor for name change")
		}

		history := &models.AMCCustomerNameHistory{
			CustomerSolutionID: cs.ID,
			CustomerID:         cs.CustomerID,
			PONumber:           cs.PONumber,
			OldName:            oldName,
			NewName:            newName,
			ChangedBy:          changedBy,
			ChangedAt:          time.Now().UTC(),
		}
		return tx.Create(history).Error
	})
}

/* =========================
   ADMIN: DELETE CONTRACT (PO)
========================= */

// DeleteCustomerSolution attempts a hard delete of the contract. If it is
// still referenced by real records (AMC assignments, tickets, notifications,
// etc.) it automatically falls back to a soft delete (is_active = false) so
// the action always succeeds, while genuinely preserving history/integrity.
// The returned bool reports whether a soft delete was performed instead.
func (s *CustomerSolutionService) DeleteCustomerSolution(id uuid.UUID) (bool, error) {
	if _, err := s.repo.GetByIDAny(id); err != nil {
		return false, err
	}

	if err := s.repo.Delete(id); err != nil {
		if utils.IsForeignKeyViolation(err) {
			if softErr := s.repo.Update(id, map[string]interface{}{"is_active": false}); softErr != nil {
				return false, softErr
			}
			return true, nil
		}
		return false, err
	}

	return false, nil
}
