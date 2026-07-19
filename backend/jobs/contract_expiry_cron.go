package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"rbac/service"
)

// ContractExpiryCron runs daily contract expiry checks and supports graceful stop.
type ContractExpiryCron struct {
	svc    *service.ContractExpiryService
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewContractExpiryCron creates a stoppable contract-expiry worker.
func NewContractExpiryCron(contractService *service.ContractExpiryService) *ContractExpiryCron {
	return &ContractExpiryCron{svc: contractService}
}

// Start begins the loop (non-blocking). Safe to call once.
func (c *ContractExpiryCron) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		log.Println("[CRON] Running initial contract expiry check...")
		c.svc.CheckAndNotifyExpiringContracts()

		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.AddDate(0, 0, 1)
			}
			duration := time.Until(next)
			log.Printf("[CRON] Next contract expiry check scheduled in %v", duration)

			timer := time.NewTimer(duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				log.Println("[CRON] Contract expiry cron stopped")
				return
			case <-timer.C:
				log.Println("[CRON] Running scheduled contract expiry check...")
				c.svc.CheckAndNotifyExpiringContracts()
			}
		}
	}()
}

// Stop cancels the loop and waits for the goroutine to exit.
func (c *ContractExpiryCron) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}

// StartContractExpiryCron starts the daily contract expiry check job (legacy helper).
func StartContractExpiryCron(contractService *service.ContractExpiryService) {
	NewContractExpiryCron(contractService).Start()
}
