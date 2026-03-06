package jobs

import (
	"time"

	"rbac/repository"
	"rbac/utils"
)

func StartEscalationCron(
	ticketRepo *repository.TicketRepository,
	escalationRepo *repository.TicketEscalationRepository,
	mailer *utils.Mailer,
) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		for {
			<-ticker.C
			EscalateOverdueTickets(
				ticketRepo,
				escalationRepo,
				mailer,
				7,
			)
		}
	}()
}

