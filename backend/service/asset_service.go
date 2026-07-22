package service

import (
	"errors"
	"strings"
	"time"

	"rbac/models"
	"rbac/repository"

	"github.com/google/uuid"
)

type AssetService struct {
	repo                 *repository.AssetRepository
	customerSolutionRepo *repository.CustomerSolutionRepository
	companyRepo          repository.CompanyRepository
}

func NewAssetService(
	repo *repository.AssetRepository,
	customerSolutionRepo *repository.CustomerSolutionRepository,
	companyRepo repository.CompanyRepository,
) *AssetService {
	return &AssetService{
		repo:                 repo,
		customerSolutionRepo: customerSolutionRepo,
		companyRepo:          companyRepo,
	}
}

type AssetInput struct {
	CompanyID          uuid.UUID
	CustomerID         *uuid.UUID
	CustomerSolutionID *uuid.UUID
	SerialNumber       string
	Name               string
	Model              string
	Category           string
	SiteLocation       string
	Notes              string
	Status             models.AssetStatus
	InstalledAt        *time.Time
}

func (s *AssetService) Create(adminID uuid.UUID, in AssetInput) (*models.Asset, error) {
	serial := strings.TrimSpace(in.SerialNumber)
	name := strings.TrimSpace(in.Name)
	if serial == "" || name == "" {
		return nil, errors.New("serial number and name are required")
	}
	if _, err := s.companyRepo.FindByID(in.CompanyID.String()); err != nil {
		return nil, errors.New("company not found")
	}

	var customerID *uuid.UUID
	if in.CustomerSolutionID != nil && *in.CustomerSolutionID != uuid.Nil {
		cs, err := s.customerSolutionRepo.GetByID(*in.CustomerSolutionID)
		if err != nil {
			return nil, errors.New("PO / customer solution not found")
		}
		customerID = &cs.CustomerID
	} else if in.CustomerID != nil {
		customerID = in.CustomerID
	}

	status := in.Status
	if status == "" {
		status = models.AssetStatusActive
	}

	now := time.Now()
	asset := &models.Asset{
		CompanyID:          in.CompanyID,
		CustomerID:         customerID,
		CustomerSolutionID: in.CustomerSolutionID,
		SerialNumber:       serial,
		Name:               name,
		Model:              strings.TrimSpace(in.Model),
		Category:           strings.TrimSpace(in.Category),
		SiteLocation:       strings.TrimSpace(in.SiteLocation),
		Notes:              strings.TrimSpace(in.Notes),
		Status:             status,
		InstalledAt:        in.InstalledAt,
		CreatedBy:          adminID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.Create(asset); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("serial number already exists")
		}
		return nil, err
	}
	return s.repo.GetByID(asset.ID)
}

func (s *AssetService) Update(id uuid.UUID, in AssetInput) (*models.Asset, error) {
	asset, err := s.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("asset not found")
	}

	serial := strings.TrimSpace(in.SerialNumber)
	name := strings.TrimSpace(in.Name)
	if serial == "" || name == "" {
		return nil, errors.New("serial number and name are required")
	}

	if in.CompanyID != uuid.Nil {
		if _, err := s.companyRepo.FindByID(in.CompanyID.String()); err != nil {
			return nil, errors.New("company not found")
		}
		asset.CompanyID = in.CompanyID
	}

	asset.SerialNumber = serial
	asset.Name = name
	asset.Model = strings.TrimSpace(in.Model)
	asset.Category = strings.TrimSpace(in.Category)
	asset.SiteLocation = strings.TrimSpace(in.SiteLocation)
	asset.Notes = strings.TrimSpace(in.Notes)
	if in.Status != "" {
		asset.Status = in.Status
	}
	asset.InstalledAt = in.InstalledAt
	asset.CustomerSolutionID = in.CustomerSolutionID

	if in.CustomerSolutionID != nil && *in.CustomerSolutionID != uuid.Nil {
		cs, err := s.customerSolutionRepo.GetByID(*in.CustomerSolutionID)
		if err != nil {
			return nil, errors.New("PO / customer solution not found")
		}
		asset.CustomerID = &cs.CustomerID
	} else {
		asset.CustomerID = in.CustomerID
	}

	asset.UpdatedAt = time.Now()
	if err := s.repo.Update(asset); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("serial number already exists")
		}
		return nil, err
	}
	return s.repo.GetByID(id)
}

func (s *AssetService) GetByID(id uuid.UUID) (*models.Asset, error) {
	return s.repo.GetByID(id)
}

func (s *AssetService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *AssetService) List(filter repository.AssetListFilter) ([]models.Asset, int64, error) {
	return s.repo.List(filter)
}
