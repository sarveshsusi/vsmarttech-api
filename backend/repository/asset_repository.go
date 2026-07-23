package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssetRepository struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

type AssetListFilter struct {
	Search             string
	CompanyID          string
	CustomerSolutionID string
	Status             string
	Statuses           []string
	// IsReplacement filters by the optional replacement checkbox when non-nil.
	IsReplacement *bool
	Limit         int
	Offset        int
}

func (r *AssetRepository) Create(asset *models.Asset) error {
	return r.db.Create(asset).Error
}

func (r *AssetRepository) Update(asset *models.Asset) error {
	return r.db.Save(asset).Error
}

func (r *AssetRepository) GetByID(id uuid.UUID) (*models.Asset, error) {
	var asset models.Asset
	err := r.db.
		Preload("Company").
		Preload("Customer").
		Preload("CustomerSolution").
		Preload("CustomerSolution.Solution").
		First(&asset, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Asset{}, "id = ?", id).Error
}

func (r *AssetRepository) List(filter AssetListFilter) ([]models.Asset, int64, error) {
	q := r.db.Model(&models.Asset{})

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where(
			"serial_number ILIKE ? OR name ILIKE ? OR model ILIKE ? OR site_location ILIKE ?",
			like, like, like, like,
		)
	}
	if filter.CompanyID != "" {
		if uid, err := uuid.Parse(filter.CompanyID); err == nil {
			q = q.Where("company_id = ?", uid)
		}
	}
	if filter.CustomerSolutionID != "" {
		if uid, err := uuid.Parse(filter.CustomerSolutionID); err == nil {
			q = q.Where("customer_solution_id = ?", uid)
		}
	}
	if len(filter.Statuses) > 0 {
		expanded := make([]string, 0, len(filter.Statuses)+3)
		for _, st := range filter.Statuses {
			expanded = append(expanded, st)
			if st == string(models.AssetStatusAtSite) {
				expanded = append(expanded,
					string(models.AssetStatusActive),
					string(models.AssetStatusInactive),
					"",
				)
			}
		}
		q = q.Where("status IN ?", expanded)
	} else if filter.Status != "" {
		if filter.Status == string(models.AssetStatusAtSite) {
			q = q.Where("status IN ?", []string{
				string(models.AssetStatusAtSite),
				string(models.AssetStatusActive),
				string(models.AssetStatusInactive),
				"",
			})
		} else {
			q = q.Where("status = ?", filter.Status)
		}
	}
	if filter.IsReplacement != nil {
		q = q.Where("is_replacement = ?", *filter.IsReplacement)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var rows []models.Asset
	err := q.
		Preload("Company").
		Preload("Customer").
		Preload("CustomerSolution").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	return rows, total, err
}

func (r *AssetRepository) CreateStatusHistory(h *models.AssetStatusHistory) error {
	return r.db.Create(h).Error
}

func (r *AssetRepository) ListStatusHistory(assetID uuid.UUID) ([]models.AssetStatusHistory, error) {
	var rows []models.AssetStatusHistory
	err := r.db.
		Where("asset_id = ?", assetID).
		Order("changed_at DESC").
		Limit(100).
		Find(&rows).Error
	return rows, err
}
