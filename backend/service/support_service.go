package service

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
	"rbac/repository"
)

type SupportService struct {
	ticketRepo          *repository.TicketRepository
	supportEngineerRepo *repository.SupportEngineerRepository
	visitRepo           *repository.ServiceVisitRepository
	feedbackRepo        *repository.FeedbackRepository
	db                  *gorm.DB
}

/* =========================
   CONSTRUCTOR
========================= */

func NewSupportService(
	ticketRepo *repository.TicketRepository,
	supportEngineerRepo *repository.SupportEngineerRepository,
	db *gorm.DB,
) *SupportService {
	return &SupportService{
		ticketRepo:          ticketRepo,
		supportEngineerRepo: supportEngineerRepo,
		visitRepo:           repository.NewServiceVisitRepository(db),
		feedbackRepo:        repository.NewFeedbackRepository(db),
		db:                  db,
	}
}

/* =========================
   ADMIN / INTERNAL USE
========================= */

func (s *SupportService) GetAssignedTickets(
	engineerID uuid.UUID,
) ([]models.Ticket, error) {
	return s.ticketRepo.GetByEngineerID(engineerID)
}

func (s *SupportService) GetMyTickets(
	userID uuid.UUID,
) ([]models.Ticket, error) {

	engineer, err := s.supportEngineerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("support engineer profile not found")
	}

	tickets, err := s.ticketRepo.GetByEngineerID(engineer.ID)
	if err != nil {
		return nil, err
	}

	if len(tickets) > 0 {
		ids := make([]string, 0, len(tickets))
		for _, t := range tickets {
			ids = append(ids, t.ID)
		}
		counts, countErr := s.visitRepo.CountByTicketIDs(ids)
		if countErr == nil {
			for i := range tickets {
				tickets[i].VisitCount = counts[tickets[i].ID]
			}
		}
		if s.feedbackRepo != nil {
			rows, fbErr := s.feedbackRepo.GetByTicketIDs(ids)
			if fbErr == nil {
				for i := range tickets {
					if fb, ok := rows[tickets[i].ID]; ok {
						summary := models.TicketFeedbackSummary{
							ID:             fb.ID,
							FeedbackStatus: fb.FeedbackStatus,
							Rating:         fb.Rating,
							Remarks:        fb.Remarks,
							SubmittedAt:    fb.SubmittedAt,
							CreatedAt:      fb.CreatedAt,
						}
						tickets[i].Feedback = &summary
					}
				}
			}
		}
	}

	return tickets, nil
}

/* =========================
   ENGINEER DASHBOARD STATS
========================= */

func (s *SupportService) GetEngineerDashboardStats(userID uuid.UUID) (map[string]interface{}, error) {
	// Get engineer profile
	engineer, err := s.supportEngineerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("support engineer profile not found")
	}

	// Get all tickets assigned to this engineer
	tickets, err := s.ticketRepo.GetByEngineerID(engineer.ID)
	if err != nil {
		return nil, err
	}

	// Calculate stats for ONLY this engineer's tickets
	totalTickets := len(tickets)
	closedTickets := 0
	pendingTickets := 0
	openTickets := 0
	inProgressTickets := 0

	for _, ticket := range tickets {
		switch ticket.Status {
		case models.StatusClosed:
			closedTickets++
		case models.StatusAssigned:
			pendingTickets++
		case models.StatusOpen:
			openTickets++
		case models.StatusInProgress:
			inProgressTickets++
		}
	}

	// Calculate rates
	closureRate := 0.0
	pendingRate := 0.0
	if totalTickets > 0 {
		closureRate = float64(closedTickets) / float64(totalTickets) * 100
		pendingRate = float64(pendingTickets) / float64(totalTickets) * 100
	}

	return map[string]interface{}{
		"total_tickets":       totalTickets,
		"closed_tickets":      closedTickets,
		"pending_tickets":     pendingTickets,
		"open_tickets":        openTickets,
		"in_progress_tickets": inProgressTickets,
		"closure_rate":        closureRate,
		"pending_rate":        pendingRate,
	}, nil
}
