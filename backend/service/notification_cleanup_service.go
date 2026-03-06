package service

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// NotificationCleanupService handles cleanup of old notifications
type NotificationCleanupService struct {
	db *gorm.DB
}

// NewNotificationCleanupService creates a new notification cleanup service
func NewNotificationCleanupService(db *gorm.DB) *NotificationCleanupService {
	return &NotificationCleanupService{
		db: db,
	}
}

// CleanupOldNotifications removes notifications older than the specified duration
func (s *NotificationCleanupService) CleanupOldNotifications(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	return s.db.Where("created_at < ?", cutoffTime).Delete(&map[string]interface{}{}).Error
}

// CleanupReadNotifications removes read notifications older than the specified duration
func (s *NotificationCleanupService) CleanupReadNotifications(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	return s.db.Where("read_at IS NOT NULL AND updated_at < ?", cutoffTime).Delete(&map[string]interface{}{}).Error
}

// ArchiveNotifications archives old notifications instead of deleting them
func (s *NotificationCleanupService) ArchiveNotifications(ctx context.Context, olderThan time.Duration) error {
	// This would be implemented based on your archive requirements
	return nil
}
