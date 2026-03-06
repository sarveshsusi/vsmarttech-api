package service

import (
	"errors"

	"github.com/google/uuid"

	"rbac/models"
	"rbac/repository"
)

type SupportService struct {
	ticketRepo          *repository.TicketRepository
	supportEngineerRepo *repository.SupportEngineerRepository
}

/* =========================
   CONSTRUCTOR
========================= */

func NewSupportService(
	ticketRepo *repository.TicketRepository,
	supportEngineerRepo *repository.SupportEngineerRepository,
) *SupportService {
	return &SupportService{
		ticketRepo:          ticketRepo,
		supportEngineerRepo: supportEngineerRepo,
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

	return s.ticketRepo.GetByEngineerID(engineer.ID)
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
