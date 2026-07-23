package service

import (
	"errors"
	"strings"

	"github.com/google/uuid"

	"rbac/models"
	"rbac/repository"
)

type FeedbackService struct {
	repo         *repository.FeedbackRepository
	ticketRepo   *repository.TicketRepository
	customerRepo *repository.CustomerRepository
}

func NewFeedbackService(
	r *repository.FeedbackRepository,
	ticketRepo *repository.TicketRepository,
	customerRepo *repository.CustomerRepository,
) *FeedbackService {
	return &FeedbackService{
		repo:         r,
		ticketRepo:   ticketRepo,
		customerRepo: customerRepo,
	}
}

func (s *FeedbackService) Submit(
	userID uuid.UUID,
	ticketID string,
	rating int,
	comment string,
) error {
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	customer, err := s.customerRepo.GetByUserID(userID)
	if err != nil {
		return errors.New("ticket not found or access denied")
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return errors.New("ticket not found or access denied")
	}

	if ticket.CustomerID != customer.ID {
		return errors.New("ticket not found or access denied")
	}

	if ticket.Status != models.StatusClosed {
		return errors.New("feedback is only allowed on closed tickets")
	}

	if ticket.EngineerID == nil {
		return errors.New("ticket has no assigned engineer")
	}

	safeComment := strings.TrimSpace(comment)
	if len(safeComment) > 2000 {
		safeComment = safeComment[:2000]
	}

	return s.repo.Create(&models.TicketFeedback{
		TicketID:   ticketID,
		EngineerID: *ticket.EngineerID,
		Rating:     rating,
		Comment:    safeComment,
	})
}
