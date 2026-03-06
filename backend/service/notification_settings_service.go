package service

import (
	"rbac/models"
	"rbac/repository"

	"gorm.io/gorm"
)

// NotificationSettingsService handles business logic for notification settings
type NotificationSettingsService struct {
	repo *repository.NotificationSettingsRepository
}

// NewNotificationSettingsService creates a new notification settings service
func NewNotificationSettingsService(db *gorm.DB) *NotificationSettingsService {
	return &NotificationSettingsService{
		repo: repository.NewNotificationSettingsRepository(db),
	}
}

// GetUserSettings retrieves notification settings for a user
func (s *NotificationSettingsService) GetUserSettings(userID string) (*models.NotificationSettings, error) {
	return s.repo.GetByUserID(userID)
}

// UpdateUserSettings updates notification settings for a user
func (s *NotificationSettingsService) UpdateUserSettings(userID string, emailNotified, smsNotified, inAppNotified bool) error {
	settings := &models.NotificationSettings{
		UserID:        userID,
		EmailNotified: emailNotified,
		SMSNotified:   smsNotified,
		InAppNotified: inAppNotified,
	}
	return s.repo.CreateOrUpdate(settings)
}

// DeleteUserSettings deletes notification settings for a user
func (s *NotificationSettingsService) DeleteUserSettings(userID string) error {
	return s.repo.Delete(userID)
}

// GetAllSettings retrieves all notification settings
func (s *NotificationSettingsService) GetAllSettings() ([]models.NotificationSettings, error) {
	return s.repo.GetAll()
}
