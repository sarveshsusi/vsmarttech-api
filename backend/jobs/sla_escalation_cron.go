package jobs

import (
	"context"
	"log"
	"time"

	"rbac/service"
	"rbac/utils"

	"gorm.io/gorm"
)

// SLAEscalationCron manages SLA escalation checks
type SLAEscalationCron struct {
	db               *gorm.DB
	slaEscalationSvc *service.SLAEscalationService
	ticker           *time.Ticker
	done             chan bool
}

// NewSLAEscalationCron creates a new SLA escalation cron job
func NewSLAEscalationCron(db *gorm.DB, mailer *utils.Mailer) *SLAEscalationCron {
	return &SLAEscalationCron{
		db:               db,
		slaEscalationSvc: service.NewSLAEscalationService(db, mailer),
		done:             make(chan bool),
	}
}

// Start starts the SLA escalation cron job
func (sec *SLAEscalationCron) Start() error {
	// Run SLA check every hour
	sec.ticker = time.NewTicker(1 * time.Hour)

	go func() {
		for {
			select {
			case <-sec.done:
				return
			case <-sec.ticker.C:
				ctx := context.Background()
				log.Println("[SLA_CRON] Starting SLA escalation check")

				if err := sec.CheckAndEscalateSLA(ctx); err != nil {
					log.Printf("[SLA_CRON_ERROR] Error checking SLA escalation: %v", err)
				}

				log.Println("[SLA_CRON] SLA escalation check completed")
			}
		}
	}()

	return nil
}

// Stop stops the SLA escalation cron job
func (sec *SLAEscalationCron) Stop() {
	if sec.ticker != nil {
		sec.ticker.Stop()
	}
	sec.done <- true
	log.Println("[SLA_CRON] SLA escalation cron job stopped")
}

// CheckAndEscalateSLA checks tickets for SLA violations and sends notifications
func (sec *SLAEscalationCron) CheckAndEscalateSLA(ctx context.Context) error {
	return sec.slaEscalationSvc.CheckAndEscalateTickets(ctx)
}

// TriggerSLACheck manually triggers the SLA check
func (sec *SLAEscalationCron) TriggerSLACheck(ctx context.Context) error {
	return sec.CheckAndEscalateSLA(ctx)
}
