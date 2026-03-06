package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"rbac/models"
	"rbac/repository"
)

type AMCAssignmentService struct {
	repo                     *repository.AMCAssignmentRepository
	notificationService      *NotificationService
	customerSolutionRepo     *repository.CustomerSolutionRepository
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

/* =========================
   ASSIGN AMC TO ENGINEER
========================= */
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
	if err := s.generateQuarterlyVisits(assignmentReq.ID, assignmentReq.AMCStartDate, assignmentReq.AMCEndDate); err != nil {
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

/* =========================
   GENERATE QUARTERLY VISITS
========================= */
func (s *AMCAssignmentService) generateQuarterlyVisits(assignmentID uuid.UUID, startDate, endDate time.Time) error {
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

		visit := &models.AMCVisit{
			AMCAssignmentID:   assignmentID,
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

/* =========================
   GET AMC DETAILS
========================= */
func (s *AMCAssignmentService) GetAMCAssignment(id uuid.UUID) (*models.AMCAssignment, error) {
	return s.repo.GetByID(id)
}

func (s *AMCAssignmentService) GetEngineerAMCs(engineerID uuid.UUID) ([]models.AMCAssignment, error) {
	return s.repo.GetByEngineer(engineerID)
}

func (s *AMCAssignmentService) GetAllAMCs() ([]models.AMCAssignment, error) {
	return s.repo.GetAll()
}

/* =========================
   COMPLETE VISIT
========================= */
func (s *AMCAssignmentService) CompleteVisit(visitID uuid.UUID, visitDate time.Time) error {
	if err := s.repo.CompleteVisit(visitID, visitDate); err != nil {
		return err
	}

	// Get visit to notify admin
	visit, err := s.repo.GetVisit(visitID)
	if err != nil {
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
   ADD PROOF/IMAGES
========================= */
func (s *AMCAssignmentService) AddVisitProof(proof *models.AMCVisitProof) error {
	if proof.AMCVisitID == uuid.Nil || proof.ImagePath == "" {
		return errors.New("visit ID and image path are required")
	}

	return s.repo.AddProof(proof)
}

func (s *AMCAssignmentService) GetVisitProofs(visitID uuid.UUID) ([]models.AMCVisitProof, error) {
	return s.repo.GetProofs(visitID)
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
