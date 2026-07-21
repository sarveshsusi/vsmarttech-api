package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
)

type ServiceVisitRepository struct {
	db *gorm.DB
}

func NewServiceVisitRepository(db *gorm.DB) *ServiceVisitRepository {
	return &ServiceVisitRepository{db: db}
}

func (r *ServiceVisitRepository) Create(visit *models.ServiceVisit) error {
	if visit.ID == uuid.Nil {
		visit.ID = uuid.New()
	}
	now := time.Now()
	visit.CreatedAt = now
	visit.UpdatedAt = now
	return r.db.Create(visit).Error
}

func (r *ServiceVisitRepository) ReplaceCoEngineers(visitID uuid.UUID, engineerIDs []uuid.UUID) error {
	visit := models.ServiceVisit{ID: visitID}
	if len(engineerIDs) == 0 {
		return r.db.Model(&visit).Association("CoEngineers").Clear()
	}

	engineers := make([]models.SupportEngineer, 0, len(engineerIDs))
	for _, id := range engineerIDs {
		engineers = append(engineers, models.SupportEngineer{ID: id})
	}
	return r.db.Model(&visit).Association("CoEngineers").Replace(engineers)
}

func (r *ServiceVisitRepository) CreateProofs(proofs []models.ServiceVisitProof) error {
	if len(proofs) == 0 {
		return nil
	}
	now := time.Now()
	for i := range proofs {
		if proofs[i].ID == uuid.Nil {
			proofs[i].ID = uuid.New()
		}
		proofs[i].CreatedAt = now
	}
	return r.db.Create(&proofs).Error
}

func (r *ServiceVisitRepository) ListByTicketID(ticketID string) ([]models.ServiceVisit, error) {
	var visits []models.ServiceVisit
	err := r.db.
		Where("ticket_id = ?", ticketID).
		Preload("Engineer.User").
		Preload("CoEngineers.User").
		Preload("Proofs").
		Order("visit_date asc, created_at asc").
		Find(&visits).Error
	return visits, err
}

func (r *ServiceVisitRepository) CountByTicketID(ticketID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.ServiceVisit{}).
		Where("ticket_id = ?", ticketID).
		Count(&count).Error
	return count, err
}

func (r *ServiceVisitRepository) CountByTicketIDs(ticketIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(ticketIDs))
	if len(ticketIDs) == 0 {
		return result, nil
	}

	type row struct {
		TicketID string
		Count    int
	}
	var rows []row
	err := r.db.Model(&models.ServiceVisit{}).
		Select("ticket_id, COUNT(*) as count").
		Where("ticket_id IN ?", ticketIDs).
		Group("ticket_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, rrow := range rows {
		result[rrow.TicketID] = rrow.Count
	}
	return result, nil
}
