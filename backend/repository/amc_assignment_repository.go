package repository

import (
	"errors"
	"time"

	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AMCAssignmentRepository struct {
	db *gorm.DB
}

func NewAMCAssignmentRepository(db *gorm.DB) *AMCAssignmentRepository {
	return &AMCAssignmentRepository{db: db}
}

/* =========================
   AMC ASSIGNMENT METHODS
========================= */

// Create assigns AMC to engineer
func (r *AMCAssignmentRepository) Create(assignment *models.AMCAssignment) error {
	assignment.ID = uuid.New()
	assignment.AssignedAt = time.Now()
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()
	return r.db.Create(assignment).Error
}

// GetByID retrieves assignment details
func (r *AMCAssignmentRepository) GetByID(id uuid.UUID) (*models.AMCAssignment, error) {
	var assignment models.AMCAssignment
	err := r.db.
		Preload("CustomerSolution").
		Preload("SupportEngineer").
		Preload("Visits", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter_start_date ASC")
		}).
		Preload("Visits.Proofs").
		Where("id = ?", id).
		First(&assignment).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("assignment not found")
		}
		return nil, err
	}

	return &assignment, nil
}

// GetByEngineer retrieves all assignments for an engineer
func (r *AMCAssignmentRepository) GetByEngineer(engineerID uuid.UUID) ([]models.AMCAssignment, error) {
	var assignments []models.AMCAssignment
	err := r.db.
		Preload("CustomerSolution").
		Preload("CustomerSolution.Customer").
		Preload("CustomerSolution.Solution").
		Preload("Visits", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter_start_date ASC")
		}).
		Preload("Visits.Proofs").
		Where("support_engineer_id = ? AND status = ?", engineerID, "active").
		Order("amc_start_date ASC").
		Find(&assignments).Error

	return assignments, err
}

// GetBySolution retrieves all assignments for a solution
func (r *AMCAssignmentRepository) GetBySolution(solutionID uuid.UUID) ([]models.AMCAssignment, error) {
	var assignments []models.AMCAssignment
	err := r.db.
		Preload("SupportEngineer").
		Preload("Visits", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter_start_date ASC")
		}).
		Preload("Visits.Proofs").
		Where("customer_solution_id = ?", solutionID).
		Order("assigned_at DESC").
		Find(&assignments).Error

	return assignments, err
}

// GetAll retrieves all active AMC assignments
func (r *AMCAssignmentRepository) GetAll() ([]models.AMCAssignment, error) {
	var assignments []models.AMCAssignment
	err := r.db.
		Preload("CustomerSolution").
		Preload("CustomerSolution.Customer").
		Preload("CustomerSolution.Solution").
		Preload("SupportEngineer").
		Preload("SupportEngineer.User").
		Preload("Visits", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter_start_date ASC")
		}).
		Preload("Visits.Proofs").
		Where("status = ?", "active").
		Order("assigned_at DESC").
		Find(&assignments).Error

	return assignments, err
}

// Update updates assignment
func (r *AMCAssignmentRepository) Update(id uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.Model(&models.AMCAssignment{}).Where("id = ?", id).Updates(updates).Error
}

// Delete removes an AMC assignment along with its visits and proofs.
// We cascade explicitly in a transaction instead of relying on a DB-level
// ON DELETE CASCADE, since the live schema is created by GORM AutoMigrate
// (see database/migrate.go) and does not carry that constraint.
func (r *AMCAssignmentRepository) Delete(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var visitIDs []uuid.UUID
		if err := tx.Model(&models.AMCVisit{}).
			Where("amc_assignment_id = ?", id).
			Pluck("id", &visitIDs).Error; err != nil {
			return err
		}

		if len(visitIDs) > 0 {
			if err := tx.Where("amc_visit_id IN ?", visitIDs).
				Delete(&models.AMCVisitProof{}).Error; err != nil {
				return err
			}

			if err := tx.Where("amc_assignment_id = ?", id).
				Delete(&models.AMCVisit{}).Error; err != nil {
				return err
			}
		}

		return tx.Delete(&models.AMCAssignment{}, "id = ?", id).Error
	})
}

/* =========================
   AMC VISIT METHODS
========================= */

// CreateVisit creates quarterly visit records
func (r *AMCAssignmentRepository) CreateVisit(visit *models.AMCVisit) error {
	visit.ID = uuid.New()
	visit.CreatedAt = time.Now()
	visit.UpdatedAt = time.Now()
	return r.db.Create(visit).Error
}

// GetVisit retrieves visit details
func (r *AMCAssignmentRepository) GetVisit(id uuid.UUID) (*models.AMCVisit, error) {
	var visit models.AMCVisit
	err := r.db.
		Preload("Proofs").
		Where("id = ?", id).
		First(&visit).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("visit not found")
		}
		return nil, err
	}

	return &visit, nil
}

// UpdateVisit updates visit details
func (r *AMCAssignmentRepository) UpdateVisit(id uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.Model(&models.AMCVisit{}).Where("id = ?", id).Updates(updates).Error
}

// ListPendingPastDue returns pending visits whose scheduled date is before endOfDay.
func (r *AMCAssignmentRepository) ListPendingPastDue(before time.Time) ([]models.AMCVisit, error) {
	var visits []models.AMCVisit
	err := r.db.
		Preload("AMCAssignment").
		Where("status = ? AND visit_scheduled_for < ?", "pending", before).
		Find(&visits).Error
	return visits, err
}

// GetVisitsByAssignment retrieves all visits for an assignment
func (r *AMCAssignmentRepository) GetVisitsByAssignment(assignmentID uuid.UUID) ([]models.AMCVisit, error) {
	var visits []models.AMCVisit
	err := r.db.
		Preload("Proofs").
		Where("amc_assignment_id = ?", assignmentID).
		Order("quarter_start_date ASC").
		Find(&visits).Error

	return visits, err
}

// DeleteNonCompletedVisits removes all pending/overdue (not-yet-completed)
// visits for an assignment — used when AMC dates change and the visit
// schedule needs to be regenerated without touching completed history.
// Cascades explicitly to any proofs uploaded against those visits, for the
// same reason Delete() does on AMCAssignmentRepository.
func (r *AMCAssignmentRepository) DeleteNonCompletedVisits(assignmentID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var visitIDs []uuid.UUID
		if err := tx.Model(&models.AMCVisit{}).
			Where("amc_assignment_id = ? AND status != ?", assignmentID, "completed").
			Pluck("id", &visitIDs).Error; err != nil {
			return err
		}

		if len(visitIDs) == 0 {
			return nil
		}

		if err := tx.Where("amc_visit_id IN ?", visitIDs).
			Delete(&models.AMCVisitProof{}).Error; err != nil {
			return err
		}

		return tx.Where("id IN ?", visitIDs).Delete(&models.AMCVisit{}).Error
	})
}

// CompleteVisit marks visit as completed
func (r *AMCAssignmentRepository) CompleteVisit(id uuid.UUID, visitDate time.Time) error {
	now := time.Now()
	return r.db.Model(&models.AMCVisit{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "completed",
			"visit_date":   visitDate,
			"completed_at": now,
			"updated_at":   now,
		}).Error
}

/* =========================
   AMC VISIT PROOF METHODS
========================= */

// AddProof adds proof image for a visit
func (r *AMCAssignmentRepository) AddProof(proof *models.AMCVisitProof) error {
	proof.ID = uuid.New()
	proof.UploadedAt = time.Now()
	return r.db.Create(proof).Error
}

// GetProofs retrieves all proofs for a visit
func (r *AMCAssignmentRepository) GetProofs(visitID uuid.UUID) ([]models.AMCVisitProof, error) {
	var proofs []models.AMCVisitProof
	err := r.db.Where("amc_visit_id = ?", visitID).Find(&proofs).Error
	return proofs, err
}

// DeleteProof removes proof image
func (r *AMCAssignmentRepository) DeleteProof(id uuid.UUID) error {
	return r.db.Delete(&models.AMCVisitProof{}, "id = ?", id).Error
}
