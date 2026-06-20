package service

import (
	"log"
	"time"

	"rbac/config"
	"rbac/repository"
	"rbac/utils"
)

type EscalationService struct {
	repo   *repository.EscalationRepository
	mailer *utils.Mailer
	cfg    *config.Config
}

func NewEscalationService(
	repo *repository.EscalationRepository,
	mailer *utils.Mailer,
	cfg *config.Config,
) *EscalationService {
	return &EscalationService{
		repo:   repo,
		mailer: mailer,
		cfg:    cfg,
	}
}

/*
	=====================
	  RUN ESCALATION CHECK

=====================
*/
func (s *EscalationService) Run() {

	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	tickets, err := s.repo.FindOverdueTickets(cutoff)
	if err != nil {
		log.Println("❌ escalation fetch failed:", err)
		return
	}

	for _, t := range tickets {

		dashboardURL := s.cfg.FrontendURL + "/dashboard/support"
		body := utils.TicketEscalationEmailTemplate(
			t.ID,
			t.Title,
			string(t.Status),
			dashboardURL,
		)

		_ = s.mailer.Send(
			"emerd@gmail.com",
			"🚨 Ticket Escalation Alert - Action Required",
			body,
		)

		_ = s.mailer.Send(
			"veemerd@gmail.com",
			"🚨 Ticket Escalation Alert - Action Required",
			body,
		)

		_ = s.repo.MarkEscalated(t.ID)
	}
}
