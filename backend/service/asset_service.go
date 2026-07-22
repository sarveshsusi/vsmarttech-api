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
	ticketRepo           *repository.TicketRepository
	customerSolutionRepo *repository.CustomerSolutionRepository
	companyRepo          repository.CompanyRepository
}

func NewAssetService(
	repo *repository.AssetRepository,
	ticketRepo *repository.TicketRepository,
	customerSolutionRepo *repository.CustomerSolutionRepository,
	companyRepo repository.CompanyRepository,
) *AssetService {
	return &AssetService{
		repo:                 repo,
		ticketRepo:           ticketRepo,
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

// LinkedTicketSummary is the open ticket currently attached to an asset.
type LinkedTicketSummary struct {
	ID     string              `json:"id"`
	Status models.TicketStatus `json:"status"`
	Title  string              `json:"title"`
}

// AssetListItem is an asset plus its newest open linked ticket (if any).
type AssetListItem struct {
	models.Asset
	LinkedTicket *LinkedTicketSummary `json:"linked_ticket,omitempty"`
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

	status := models.NormalizeAssetStatus(in.Status)
	if in.Status != "" && !models.IsValidAssetStatus(in.Status) {
		return nil, errors.New("invalid asset status")
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
		if !models.IsValidAssetStatus(in.Status) {
			return nil, errors.New("invalid asset status")
		}
		asset.Status = models.NormalizeAssetStatus(in.Status)
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

func (s *AssetService) UpdateStatus(id uuid.UUID, status models.AssetStatus) (*models.Asset, error) {
	if !models.IsValidAssetStatus(status) {
		return nil, errors.New("invalid asset status")
	}
	asset, err := s.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("asset not found")
	}
	asset.Status = models.NormalizeAssetStatus(status)
	asset.UpdatedAt = time.Now()
	if err := s.repo.Update(asset); err != nil {
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

func (s *AssetService) List(filter repository.AssetListFilter) ([]AssetListItem, int64, error) {
	rows, total, err := s.repo.List(filter)
	if err != nil {
		return nil, 0, err
	}

	ids := make([]uuid.UUID, 0, len(rows))
	for _, a := range rows {
		ids = append(ids, a.ID)
	}
	byAsset, err := s.ticketRepo.FindLatestOpenByAssetIDs(ids)
	if err != nil {
		return nil, 0, err
	}

	out := make([]AssetListItem, 0, len(rows))
	for _, a := range rows {
		item := AssetListItem{Asset: a}
		if t, ok := byAsset[a.ID]; ok {
			item.LinkedTicket = &LinkedTicketSummary{
				ID:     t.ID,
				Status: t.Status,
				Title:  t.Title,
			}
		}
		out = append(out, item)
	}
	return out, total, nil
}

// LinkTicket attaches an open ticket to an asset (sets tickets.asset_id).
// Empty ticketID clears the link on any open ticket currently pointing at this asset.
func (s *AssetService) LinkTicket(assetID uuid.UUID, ticketID string) (*LinkedTicketSummary, error) {
	asset, err := s.repo.GetByID(assetID)
	if err != nil {
		return nil, errors.New("asset not found")
	}

	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		if err := s.ticketRepo.ClearAssetLinks(assetID); err != nil {
			return nil, err
		}
		return nil, nil
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}
	if ticket.Status == models.StatusClosed {
		return nil, errors.New("cannot link a closed ticket")
	}
	if ticket.Customer.CompanyID != asset.CompanyID {
		return nil, errors.New("ticket company does not match asset company")
	}

	if err := s.ticketRepo.ClearAssetLinks(assetID); err != nil {
		return nil, err
	}

	if err := s.ticketRepo.UpdateFields(ticketID, map[string]interface{}{
		"asset_id": assetID,
	}); err != nil {
		return nil, err
	}

	return &LinkedTicketSummary{
		ID:     ticket.ID,
		Status: ticket.Status,
		Title:  ticket.Title,
	}, nil
}

// ListOpenTicketsForCompany returns non-closed tickets for the link-ticket dropdown.
func (s *AssetService) ListOpenTicketsForCompany(companyID uuid.UUID) ([]LinkedTicketSummary, error) {
	if _, err := s.companyRepo.FindByID(companyID.String()); err != nil {
		return nil, errors.New("company not found")
	}
	tickets, err := s.ticketRepo.ListOpenByCompanyID(companyID)
	if err != nil {
		return nil, err
	}
	out := make([]LinkedTicketSummary, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, LinkedTicketSummary{
			ID:     t.ID,
			Status: t.Status,
			Title:  t.Title,
		})
	}
	return out, nil
}
