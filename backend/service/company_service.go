package service

import (
	"errors"
	"strings"

	"rbac/models"
	"rbac/repository"

	"github.com/google/uuid"
)

type CompanyService interface {
	CreateCompany(name string) (*models.Company, error)
	GetCompanies() ([]models.Company, error)
	UpdateCompany(id string, name string) (*models.Company, error)
	DeleteCompany(id string) error
}

type companyService struct {
	repo repository.CompanyRepository
}

func NewCompanyService(repo repository.CompanyRepository) CompanyService {
	return &companyService{repo}
}

func (s *companyService) CreateCompany(name string) (*models.Company, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("company name is required")
	}

	company := &models.Company{
		ID:   uuid.New(),
		Name: name,
	}

	if err := s.repo.Create(company); err != nil {
		return nil, err
	}

	return company, nil
}

func (s *companyService) GetCompanies() ([]models.Company, error) {
	return s.repo.FindAll()
}

func (s *companyService) UpdateCompany(id string, name string) (*models.Company, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("company name is required")
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid company id")
	}

	company := &models.Company{
		ID:   uid,
		Name: name,
	}

	if err := s.repo.Update(company); err != nil {
		return nil, err
	}

	return company, nil
}

func (s *companyService) DeleteCompany(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid company id")
	}

	return s.repo.Delete(uid)
}
