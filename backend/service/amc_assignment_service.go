package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"rbac/models"
	"rbac/repository"
)

type AMCAssignmentService struct {
	repo                 *repository.AMCAssignmentRepository
	notificationService  *NotificationService
	customerSolutionRepo *repository.CustomerSolutionRepository
}

func NewAMCAssignmentService(
	repo *repository.AMCAssignmentRepository,
	notificationService *NotificationService,
	customerSolutionRepo *repository.CustomerSolutionRepository,
) *AMCAssignmentService {
	return &AMCAssignmentService{
		repo:                 repo,
		notificationService:  notificationService,
		customerSolutionRepo: customerSolutionRepo,
	}
}

/*
	=========================
	  ASSIGN AMC TO ENGINEER

=========================
*/
func (s *AMCAssignmentService) AssignAMC(assignmentReq *models.AMCAssignment) error {
	if assignmentReq.CustomerSolutionID == uuid.Nil || assignmentReq.SupportEngineerID == uuid.Nil {
		return errors.New("customer solution and engineer are required")
	}

	if assignmentReq.AMCStartDate.After(assignmentReq.AMCEndDate) {
		return errors.New("AMC start date must be before end date")
	}

	// Create assignment
	if err := s.repo.Create(assignmentReq); err != nil {
		return err
	}

	// Generate quarterly visits based on AMC dates
	if err := s.generateQuarterlyVisits(
		assignmentReq.ID,
		assignmentReq.SupportEngineerID,
		assignmentReq.AMCStartDate,
		assignmentReq.AMCEndDate,
	); err != nil {
		return err
	}

	// Send notification to engineer
	s.notificationService.NotifyAMCAssigned(
		assignmentReq.SupportEngineerID,
		assignmentReq.ID,
		assignmentReq.AMCStartDate,
		assignmentReq.AMCEndDate,
	)

	return nil
}

/*
	=========================
	  GENERATE QUARTERLY VISITS

=========================
*/
func (s *AMCAssignmentService) generateQuarterlyVisits(assignmentID, engineerID uuid.UUID, startDate, endDate time.Time) error {
	currentDate := startDate

	// Generate visits for each quarter
	for currentDate.Before(endDate) {
		// Calculate quarter end
		quarterEnd := getQuarterEnd(currentDate)
		if quarterEnd.After(endDate) {
			quarterEnd = endDate
		}

		// Calculate visit scheduled date (middle of quarter or start + 1 month)
		visitScheduled := currentDate.AddDate(0, 1, 0) // 1 month into quarter

		engID := engineerID
		visit := &models.AMCVisit{
			AMCAssignmentID:   assignmentID,
			SupportEngineerID: &engID,
			QuarterStartDate:  currentDate,
			QuarterEndDate:    quarterEnd,
			VisitScheduledFor: visitScheduled,
			Status:            "pending",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		if err := s.repo.CreateVisit(visit); err != nil {
			return err
		}

		// Move to next quarter
		currentDate = quarterEnd.AddDate(0, 0, 1)
	}

	return nil
}

/*
	=========================
	  GET AMC DETAILS

=========================
*/
func (s *AMCAssignmentService) GetAMCAssignment(id uuid.UUID) (*models.AMCAssignment, error) {
	return s.repo.GetByID(id)
}

func (s *AMCAssignmentService) GetEngineerAMCs(engineerID uuid.UUID) ([]models.AMCAssignment, error) {
	return s.repo.GetByEngineer(engineerID)
}

func (s *AMCAssignmentService) GetAllAMCs() ([]models.AMCAssignment, error) {
	return s.repo.GetAll()
}

func (s *AMCAssignmentService) GetCustomerAMCs(customerID uuid.UUID) ([]models.AMCAssignment, error) {
	return s.repo.GetByCustomer(customerID)
}

/*
	=========================
	  COMPLETE VISIT

=========================
*/
func (s *AMCAssignmentService) CompleteVisit(visitID uuid.UUID, visitDate time.Time) error {
	visit, err := s.repo.GetVisit(visitID)
	if err != nil {
		return err
	}
	if err := errIfAMCClosed(visit.AMCAssignment); err != nil {
		return err
	}

	engineerID := visit.SupportEngineerID
	if engineerID == nil && visit.AMCAssignment != nil {
		id := visit.AMCAssignment.SupportEngineerID
		engineerID = &id
	}

	if err := s.repo.CompleteVisit(visitID, visitDate, engineerID); err != nil {
		return err
	}

	// Notify admin
	s.notificationService.NotifyVisitCompleted(
		visit.AMCAssignmentID,
		visitID,
		visitDate,
	)

	return nil
}

/* =========================
   OVERDUE + RESCHEDULE
========================= */

// MarkOverdueVisits sets pending visits past their scheduled date to overdue and notifies engineers.
func (s *AMCAssignmentService) MarkOverdueVisits() (int, error) {
	now := time.Now()
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	visits, err := s.repo.ListPendingPastDue(endOfToday)
	if err != nil {
		return 0, err
	}

	marked := 0
	for _, visit := range visits {
		if err := s.repo.UpdateVisit(visit.ID, map[string]interface{}{
			"status": "overdue",
		}); err != nil {
			continue
		}
		marked++
		if s.notificationService != nil && visit.AMCAssignment != nil {
			s.notificationService.NotifyVisitOverdue(
				visit.AMCAssignment.SupportEngineerID,
				visit.ID,
				visit.VisitScheduledFor,
			)
		}
	}
	return marked, nil
}

// RescheduleVisit moves a pending/overdue visit to a new scheduled date (back to pending).
func (s *AMCAssignmentService) RescheduleVisit(visitID uuid.UUID, scheduledFor time.Time) (*models.AMCVisit, error) {
	visit, err := s.repo.GetVisit(visitID)
	if err != nil {
		return nil, err
	}
	if err := errIfAMCClosed(visit.AMCAssignment); err != nil {
		return nil, err
	}
	if visit.Status == "completed" {
		return nil, errors.New("cannot reschedule a completed visit")
	}
	if err := s.repo.UpdateVisit(visitID, map[string]interface{}{
		"visit_scheduled_for": scheduledFor,
		"status":              "pending",
	}); err != nil {
		return nil, err
	}
	return s.repo.GetVisit(visitID)
}

/*
	=========================
	  ADD PROOF/IMAGES

=========================
*/
func (s *AMCAssignmentService) AddVisitProof(proof *models.AMCVisitProof) error {
	if proof.AMCVisitID == uuid.Nil || proof.ImagePath == "" {
		return errors.New("visit ID and image path are required")
	}

	visit, err := s.repo.GetVisit(proof.AMCVisitID)
	if err != nil {
		return err
	}
	if err := errIfAMCClosed(visit.AMCAssignment); err != nil {
		return err
	}

	return s.repo.AddProof(proof)
}

func (s *AMCAssignmentService) GetVisitProofs(visitID uuid.UUID) ([]models.AMCVisitProof, error) {
	return s.repo.GetProofs(visitID)
}

func (s *AMCAssignmentService) GetProof(id uuid.UUID) (*models.AMCVisitProof, error) {
	return s.repo.GetProof(id)
}

var (
	ErrProofNotFound  = errors.New("proof not found")
	ErrProofForbidden = errors.New("not authorized to modify this proof")
	ErrProofLocked    = errors.New("cannot modify proofs on a completed visit")
)

func (s *AMCAssignmentService) loadMutableOwnedProof(proofID, engineerID uuid.UUID) (*models.AMCVisitProof, error) {
	proof, err := s.repo.GetProof(proofID)
	if err != nil {
		if err.Error() == "proof not found" {
			return nil, ErrProofNotFound
		}
		return nil, err
	}

	visit, err := s.repo.GetVisit(proof.AMCVisitID)
	if err != nil {
		return nil, err
	}
	if err := errIfAMCClosed(visit.AMCAssignment); err != nil {
		return nil, err
	}
	if visit.Status == "completed" {
		return nil, ErrProofLocked
	}
	if visit.AMCAssignment == nil || visit.AMCAssignment.SupportEngineerID != engineerID {
		return nil, ErrProofForbidden
	}

	return proof, nil
}

func (s *AMCAssignmentService) UpdateVisitProof(proofID, engineerID uuid.UUID, imageURL, description *string) (*models.AMCVisitProof, error) {
	if _, err := s.loadMutableOwnedProof(proofID, engineerID); err != nil {
		return nil, err
	}
	if imageURL == nil && description == nil {
		return nil, errors.New("no updates provided")
	}

	updates := map[string]interface{}{}
	if imageURL != nil {
		if strings.TrimSpace(*imageURL) == "" {
			return nil, errors.New("image path is required")
		}
		updates["image_path"] = *imageURL
		updates["uploaded_at"] = time.Now()
	}
	if description != nil {
		updates["description"] = *description
	}

	if err := s.repo.UpdateProof(proofID, updates); err != nil {
		return nil, err
	}
	return s.repo.GetProof(proofID)
}

func (s *AMCAssignmentService) DeleteVisitProof(proofID, engineerID uuid.UUID) error {
	if _, err := s.loadMutableOwnedProof(proofID, engineerID); err != nil {
		return err
	}
	return s.repo.DeleteProof(proofID)
}

/* =========================
   UPDATE AMC ASSIGNMENT
========================= */

type UpdateAMCAssignmentRequest struct {
	SupportEngineerID *uuid.UUID
	AMCStartDate      *time.Time
	AMCEndDate        *time.Time
	Status            *string
	Notes             *string
}

func (s *AMCAssignmentService) UpdateAMCAssignment(id uuid.UUID, req *UpdateAMCAssignmentRequest) error {
	assignment, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if assignmentStatus(assignment) == "closed" {
		if req.Status != nil && strings.ToLower(strings.TrimSpace(*req.Status)) != "closed" {
			return errors.New("closed AMC cannot be edited")
		}
		if req.SupportEngineerID != nil && *req.SupportEngineerID != assignment.SupportEngineerID {
			return errors.New("closed AMC cannot be edited")
		}
		if req.AMCStartDate != nil && !req.AMCStartDate.Equal(assignment.AMCStartDate) {
			return errors.New("closed AMC cannot be edited")
		}
		if req.AMCEndDate != nil && !req.AMCEndDate.Equal(assignment.AMCEndDate) {
			return errors.New("closed AMC cannot be edited")
		}
	}

	if req.Status != nil {
		switch strings.ToLower(strings.TrimSpace(*req.Status)) {
		case "active", "completed", "expired", "closed":
		default:
			return errors.New("invalid status")
		}
		normalized := strings.ToLower(strings.TrimSpace(*req.Status))
		req.Status = &normalized
		if normalized == "closed" && !AMCAssignmentCanClose(assignment, time.Now()) {
			return errors.New("AMC can only be closed after it has expired")
		}
	}

	hasCompletedVisit := false
	for _, v := range assignment.Visits {
		if v.Status == "completed" {
			hasCompletedVisit = true
			break
		}
	}

	updates := map[string]interface{}{}
	engineerChanged := false

	if req.SupportEngineerID != nil && *req.SupportEngineerID != uuid.Nil {
		if *req.SupportEngineerID != assignment.SupportEngineerID {
			engineerChanged = true
			updates["assigned_at"] = time.Now()
		}
		updates["support_engineer_id"] = *req.SupportEngineerID
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}

	newStart := assignment.AMCStartDate
	newEnd := assignment.AMCEndDate
	datesChanged := false

	if req.AMCStartDate != nil && !req.AMCStartDate.Equal(assignment.AMCStartDate) {
		if hasCompletedVisit {
			return errors.New("cannot change start date after visits have already been completed")
		}
		newStart = *req.AMCStartDate
		updates["amc_start_date"] = newStart
		datesChanged = true
	}

	if req.AMCEndDate != nil && !req.AMCEndDate.Equal(assignment.AMCEndDate) {
		newEnd = *req.AMCEndDate
		updates["amc_end_date"] = newEnd
		datesChanged = true
	}

	if datesChanged && newStart.After(newEnd) {
		return errors.New("AMC start date must be before end date")
	}

	if len(updates) > 0 {
		if err := s.repo.Update(id, updates); err != nil {
			return err
		}
	}

	if datesChanged {
		engineerID := assignment.SupportEngineerID
		if req.SupportEngineerID != nil && *req.SupportEngineerID != uuid.Nil {
			engineerID = *req.SupportEngineerID
		}
		if err := s.regenerateNonCompletedVisits(id, engineerID, assignment.Visits, newStart, newEnd); err != nil {
			return err
		}
	} else if engineerChanged {
		if err := s.repo.UpdateOpenVisitEngineer(id, *req.SupportEngineerID); err != nil {
			return err
		}
	}

	if engineerChanged {
		s.notificationService.NotifyAMCAssigned(
			*req.SupportEngineerID,
			id,
			newStart,
			newEnd,
		)
	}

	return nil
}

// regenerateNonCompletedVisits re-derives the pending quarterly visit
// schedule after a date change, without touching visits already completed.
func (s *AMCAssignmentService) regenerateNonCompletedVisits(
	assignmentID uuid.UUID,
	engineerID uuid.UUID,
	existingVisits []models.AMCVisit,
	startDate, endDate time.Time,
) error {
	// Resume scheduling right after the latest completed quarter, so history
	// is preserved; fall back to the (possibly new) start date otherwise.
	regenFrom := startDate
	for _, v := range existingVisits {
		if v.Status == "completed" && v.QuarterEndDate.AddDate(0, 0, 1).After(regenFrom) {
			regenFrom = v.QuarterEndDate.AddDate(0, 0, 1)
		}
	}

	if err := s.repo.DeleteNonCompletedVisits(assignmentID); err != nil {
		return err
	}

	if regenFrom.After(endDate) {
		return nil // contract shortened past the last completed visit — nothing left to schedule
	}

	return s.generateQuarterlyVisits(assignmentID, engineerID, regenFrom, endDate)
}

/* =========================
   DELETE AMC ASSIGNMENT
========================= */

func (s *AMCAssignmentService) DeleteAMCAssignment(id uuid.UUID) error {
	if _, err := s.repo.GetByID(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

/* =========================
   HELPER FUNCTIONS
========================= */

// getQuarterEnd returns the last day of the quarter
func getQuarterEnd(date time.Time) time.Time {
	month := date.Month()
	year := date.Year()

	var endMonth, endDay int

	switch {
	case month <= 3:
		endMonth, endDay = 3, 31
	case month <= 6:
		endMonth, endDay = 6, 30
	case month <= 9:
		endMonth, endDay = 9, 30
	default:
		endMonth, endDay = 12, 31
	}

	return time.Date(year, time.Month(endMonth), endDay, 23, 59, 59, 0, date.Location())
}

// GetQuarterInfo returns quarter details from a date
func GetQuarterInfo(date time.Time) (start time.Time, end time.Time, quarterNum int) {
	month := date.Month()
	year := date.Year()

	switch {
	case month <= 3:
		start = time.Date(year, time.January, 1, 0, 0, 0, 0, date.Location())
		end = time.Date(year, time.March, 31, 23, 59, 59, 0, date.Location())
		quarterNum = 1
	case month <= 6:
		start = time.Date(year, time.April, 1, 0, 0, 0, 0, date.Location())
		end = time.Date(year, time.June, 30, 23, 59, 59, 0, date.Location())
		quarterNum = 2
	case month <= 9:
		start = time.Date(year, time.July, 1, 0, 0, 0, 0, date.Location())
		end = time.Date(year, time.September, 30, 23, 59, 59, 0, date.Location())
		quarterNum = 3
	default:
		start = time.Date(year, time.October, 1, 0, 0, 0, 0, date.Location())
		end = time.Date(year, time.December, 31, 23, 59, 59, 0, date.Location())
		quarterNum = 4
	}

	return
}

// GetQuarterLabel returns human-readable quarter label
func GetQuarterLabel(quarterNum int, year int) string {
	quarters := map[int]string{
		1: "Q1 (Jan-Mar)",
		2: "Q2 (Apr-Jun)",
		3: "Q3 (Jul-Sep)",
		4: "Q4 (Oct-Dec)",
	}
	return fmt.Sprintf("%s %d", quarters[quarterNum], year)
}
