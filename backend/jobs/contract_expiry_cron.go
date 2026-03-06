package jobs

import (
	"log"
	"time"

	"rbac/service"
)

// StartContractExpiryCron starts the daily contract expiry check job
func StartContractExpiryCron(contractService *service.ContractExpiryService) {
	go func() {
		// Run immediately on startup
		log.Println("[CRON] Running initial contract expiry check...")
		contractService.CheckAndNotifyExpiringContracts()

		// Then run daily at 9 AM
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())

			// If it's already past 9 AM, schedule for tomorrow
			if now.After(next) {
				next = next.AddDate(0, 0, 1)
			}

			duration := time.Until(next)
			log.Printf("[CRON] Next contract expiry check scheduled in %v", duration)

			time.Sleep(duration)

			log.Println("[CRON] Running scheduled contract expiry check...")
			contractService.CheckAndNotifyExpiringContracts()
		}
	}()
}
