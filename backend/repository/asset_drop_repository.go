package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssetDropRepository struct {
	db *gorm.DB
}

func NewAssetDropRepository(db *gorm.DB) *AssetDropRepository {
	return &AssetDropRepository{db: db}
}

func (r *AssetDropRepository) Create(drop *models.TicketAssetDrop) error {
	return r.db.Create(drop).Error
}

func (r *AssetDropRepository) Update(drop *models.TicketAssetDrop) error {
	return r.db.Save(drop).Error
}

func (r *AssetDropRepository) GetByID(id uuid.UUID) (*models.TicketAssetDrop, error) {
	var drop models.TicketAssetDrop
	err := r.db.
		Preload("ReturnEngineer").
		Preload("ReturnEngineer.User").
		Preload("Asset").
		Preload("Ticket").
		First(&drop, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &drop, nil
}

func (r *AssetDropRepository) GetActiveByAssetID(assetID uuid.UUID) (*models.TicketAssetDrop, error) {
	var drop models.TicketAssetDrop
	err := r.db.
		Preload("ReturnEngineer").
		Preload("ReturnEngineer.User").
		Preload("Asset").
		Where(
			"asset_id = ? AND status IN ?",
			assetID,
			[]models.AssetDropStatus{
				models.AssetDropStatusRequested,
				models.AssetDropStatusAcknowledged,
				models.AssetDropStatusReturnAssigned,
			},
		).
		Order("created_at DESC").
		First(&drop).Error
	if err != nil {
		return nil, err
	}
	return &drop, nil
}

func (r *AssetDropRepository) GetLatestByAssetID(assetID uuid.UUID) (*models.TicketAssetDrop, error) {
	var drop models.TicketAssetDrop
	err := r.db.
		Preload("Asset").
		Where("asset_id = ?", assetID).
		Order("created_at DESC").
		First(&drop).Error
	if err != nil {
		return nil, err
	}
	return &drop, nil
}

func (r *AssetDropRepository) GetActiveByTicketID(ticketID string) (*models.TicketAssetDrop, error) {
	var drop models.TicketAssetDrop
	err := r.db.
		Preload("ReturnEngineer").
		Preload("ReturnEngineer.User").
		Preload("Asset").
		Where(
			"ticket_id = ? AND status IN ?",
			ticketID,
			[]models.AssetDropStatus{
				models.AssetDropStatusRequested,
				models.AssetDropStatusAcknowledged,
				models.AssetDropStatusReturnAssigned,
			},
		).
		Order("created_at DESC").
		First(&drop).Error
	if err != nil {
		return nil, err
	}
	return &drop, nil
}

func (r *AssetDropRepository) GetLatestByTicketID(ticketID string) (*models.TicketAssetDrop, error) {
	var drop models.TicketAssetDrop
	err := r.db.
		Preload("ReturnEngineer").
		Preload("ReturnEngineer.User").
		Preload("Asset").
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		First(&drop).Error
	if err != nil {
		return nil, err
	}
	return &drop, nil
}

func (r *AssetDropRepository) ListLatestByTicketIDs(ticketIDs []string) (map[string]models.TicketAssetDrop, error) {
	result := make(map[string]models.TicketAssetDrop)
	if len(ticketIDs) == 0 {
		return result, nil
	}

	var drops []models.TicketAssetDrop
	err := r.db.
		Preload("ReturnEngineer").
		Preload("ReturnEngineer.User").
		Preload("Asset").
		Where("ticket_id IN ?", ticketIDs).
		Order("created_at DESC").
		Find(&drops).Error
	if err != nil {
		return nil, err
	}
	for _, d := range drops {
		if _, exists := result[d.TicketID]; exists {
			continue
		}
		result[d.TicketID] = d
	}
	return result, nil
}

func (r *AssetDropRepository) ListAdmin(statuses []models.AssetDropStatus) ([]models.TicketAssetDrop, error) {
	q := r.db.
		Preload("ReturnEngineer").
		Preload("ReturnEngineer.User").
		Preload("Asset").
		Preload("Ticket").
		Preload("Ticket.Customer").
		Preload("Ticket.Customer.Company").
		Preload("Ticket.CustomerSolution").
		Preload("Ticket.SupportEngineer").
		Preload("Ticket.SupportEngineer.User")

	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	} else {
		q = q.Where("status IN ?", []models.AssetDropStatus{
			models.AssetDropStatusRequested,
			models.AssetDropStatusAcknowledged,
			models.AssetDropStatusReturnAssigned,
		})
	}

	var drops []models.TicketAssetDrop
	err := q.Order("created_at DESC").Find(&drops).Error
	return drops, err
}

func (r *AssetDropRepository) ListForSupportEngineer(engineerID uuid.UUID) ([]models.TicketAssetDrop, error) {
	var drops []models.TicketAssetDrop
	err := r.db.
		Preload("ReturnEngineer").
		Preload("ReturnEngineer.User").
		Preload("Asset").
		Preload("Ticket").
		Preload("Ticket.Customer").
		Preload("Ticket.Customer.Company").
		Where(
			`(
				ticket_id IN (SELECT id FROM tickets WHERE engineer_id = ?)
				OR return_engineer_id = ?
			) AND status <> ?`,
			engineerID,
			engineerID,
			models.AssetDropStatusReturned,
		).
		Order("created_at DESC").
		Find(&drops).Error
	return drops, err
}

func (r *AssetDropRepository) ListReturnAssignments(engineerID uuid.UUID) ([]models.TicketAssetDrop, error) {
	var drops []models.TicketAssetDrop
	err := r.db.
		Preload("Asset").
		Preload("Ticket").
		Preload("Ticket.Customer").
		Preload("Ticket.Customer.Company").
		Where("return_engineer_id = ? AND status = ?", engineerID, models.AssetDropStatusReturnAssigned).
		Order("return_assigned_at DESC NULLS LAST, created_at DESC").
		Find(&drops).Error
	return drops, err
}
