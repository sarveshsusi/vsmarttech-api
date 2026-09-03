package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"rbac/repository"
)

const auditCleanupHour = 3 // 03:00 local time on the 1st of each month

// AuditLogCleanupCron deletes HTTP audit rows older than the configured
// retention on startup and again at the start of each month.
type AuditLogCleanupCron struct {
	repo      *repository.AuditRepository
	retention time.Duration
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewAuditLogCleanupCron(repo *repository.AuditRepository, retentionDays int) *AuditLogCleanupCron {
	if retentionDays < 1 {
		retentionDays = 30
	}
	return &AuditLogCleanupCron{
		repo:      repo,
		retention: time.Duration(retentionDays) * 24 * time.Hour,
	}
}

func (c *AuditLogCleanupCron) Start() {
	if c.repo == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		log.Println("[CRON] Running initial audit log cleanup...")
		c.runOnce()

		for {
			now := time.Now()
			next := nextMonthStart(now, auditCleanupHour)
			log.Printf("[CRON] Next audit log cleanup scheduled in %v (at %s)", time.Until(next).Round(time.Minute), next.Format(time.RFC3339))

			timer := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				timer.Stop()
				log.Println("[CRON] Audit log cleanup cron stopped")
				return
			case <-timer.C:
				log.Println("[CRON] Running monthly audit log cleanup...")
				c.runOnce()
			}
		}
	}()
}

func (c *AuditLogCleanupCron) runOnce() {
	cutoff := time.Now().Add(-c.retention)
	deleted, err := c.repo.DeleteOlderThan(cutoff)
	if err != nil {
		log.Printf("[CRON] Audit log cleanup failed: %v", err)
		return
	}
	log.Printf("[CRON] Deleted %d audit logs older than %s", deleted, cutoff.Format(time.RFC3339))
	if deleted == 0 {
		return
	}
	if err := c.repo.VacuumAuditLogs(); err != nil {
		log.Printf("[CRON] VACUUM audit_logs skipped: %v", err)
	}
}

func (c *AuditLogCleanupCron) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}

// nextMonthStart returns the next 1st-of-month at the given hour in now's location.
func nextMonthStart(now time.Time, hour int) time.Time {
	year, month, _ := now.Date()
	next := time.Date(year, month, 1, hour, 0, 0, 0, now.Location())
	if !now.Before(next) {
		next = next.AddDate(0, 1, 0)
	}
	return next
}
