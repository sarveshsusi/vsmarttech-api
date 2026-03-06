package jobs

import (
	"fmt"
	"time"

	"rbac/repository"
	"rbac/utils"
)

func EscalateOverdueTickets(
	ticketRepo *repository.TicketRepository,
	escalationRepo *repository.TicketEscalationRepository,
	mailer *utils.Mailer,
	days int,
) {

	tickets, err := ticketRepo.FindOverdueTickets(days)
	if err != nil {
		return // never panic in background jobs
	}

	recipients := []string{
		"emerd@gmail.com",
		"veemerd@gmail.com",
	}

	for _, t := range tickets {

		already, err := escalationRepo.AlreadyEscalated(t.ID)
		if err != nil || already {
			continue
		}

		body := fmt.Sprintf(`
🚨 Ticket Escalation Alert

Ticket ID: %s
Title: %s
Status: %s
Priority: %s
Created At: %s
Days Open: %d

This ticket has exceeded SLA and requires immediate attention.
`,
			t.ID,
			t.Title,
			t.Status,
			t.Priority,
			t.CreatedAt.Format(time.RFC3339),
			days,
		)

		for _, email := range recipients {
			_ = mailer.Send(
				email,
				"🚨 Ticket Escalation – SLA Breach",
				body,
			)
		}

		_ = escalationRepo.Create(t.ID)
	}
}
