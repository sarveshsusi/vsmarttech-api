package service

import (
	"github.com/google/uuid"

	"rbac/models"
	"rbac/repository"
)

type FeedbackService struct {
	repo *repository.FeedbackRepository
}

func NewFeedbackService(r *repository.FeedbackRepository) *FeedbackService {
	return &FeedbackService{repo: r}
}

func (s *FeedbackService) Submit(
	ticketID uuid.UUID,
	engineerID uuid.UUID,
	rating int,
	comment string,
) error {

	return s.repo.Create(&models.TicketFeedback{
		TicketID:   ticketID,
		EngineerID: engineerID,
		Rating:    rating,
		Comment:   comment,
	})
}
