package jobs

import (
	"context"
	"log"
	"rbac/service"
	"time"

	"gorm.io/gorm"
)

// NotificationCleanupCron manages scheduled notification cleanup
type NotificationCleanupCron struct {
	service *service.NotificationCleanupService
	ticker  *time.Ticker
	done    chan bool
}

// NewNotificationCleanupCron creates a new notification cleanup cron job
func NewNotificationCleanupCron(db *gorm.DB) *NotificationCleanupCron {
	return &NotificationCleanupCron{
		service: service.NewNotificationCleanupService(db),
		done:    make(chan bool),
	}
}

// Start starts the notification cleanup cron job
func (nc *NotificationCleanupCron) Start() error {
	// Run cleanup job every 24 hours
	nc.ticker = time.NewTicker(24 * time.Hour)

	go func() {
		for {
			select {
			case <-nc.done:
				return
			case <-nc.ticker.C:
				ctx := context.Background()
				log.Println("Starting notification cleanup job")

				// Clean up notifications older than 30 days
				if err := nc.service.CleanupOldNotifications(ctx, 30*24*time.Hour); err != nil {
					log.Printf("Error cleaning up old notifications: %v", err)
				}

				// Clean up read notifications older than 7 days
				if err := nc.service.CleanupReadNotifications(ctx, 7*24*time.Hour); err != nil {
					log.Printf("Error cleaning up read notifications: %v", err)
				}

				log.Println("Notification cleanup job completed")
			}
		}
	}()

	return nil
}

// Stop stops the notification cleanup cron job
func (nc *NotificationCleanupCron) Stop() {
	if nc.ticker != nil {
		nc.ticker.Stop()
	}
	nc.done <- true
	log.Println("Notification cleanup cron job stopped")
}

// TriggerCleanup manually triggers the cleanup job
func (nc *NotificationCleanupCron) TriggerCleanup(ctx context.Context) error {
	if err := nc.service.CleanupOldNotifications(ctx, 30*24*time.Hour); err != nil {
		return err
	}
	return nc.service.CleanupReadNotifications(ctx, 7*24*time.Hour)
}
