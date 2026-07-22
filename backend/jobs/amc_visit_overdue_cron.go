package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"rbac/service"
)

// AMCVisitOverdueCron marks pending AMC visits past schedule as overdue (daily).
type AMCVisitOverdueCron struct {
	svc    *service.AMCAssignmentService
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewAMCVisitOverdueCron(svc *service.AMCAssignmentService) *AMCVisitOverdueCron {
	return &AMCVisitOverdueCron{svc: svc}
}

func (c *AMCVisitOverdueCron) Start() {
	if c.svc == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		log.Println("[CRON] Running initial AMC visit overdue check...")
		c.runOnce()

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.AddDate(0, 0, 1)
			}
			timer := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				timer.Stop()
				log.Println("[CRON] AMC visit overdue cron stopped")
				return
			case <-timer.C:
				log.Println("[CRON] Running AMC visit overdue check...")
				c.runOnce()
			}
		}
	}()
}

func (c *AMCVisitOverdueCron) runOnce() {
	n, err := c.svc.MarkOverdueVisits()
	if err != nil {
		log.Printf("[CRON] AMC overdue check failed: %v", err)
		return
	}
	log.Printf("[CRON] Marked %d AMC visits overdue", n)
}

func (c *AMCVisitOverdueCron) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}
