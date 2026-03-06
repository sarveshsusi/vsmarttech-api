package repository

import (
	"rbac/models"

	"gorm.io/gorm"
)

// NotificationSettingsRepository handles database operations for notification settings
type NotificationSettingsRepository struct {
	db *gorm.DB
}

// NewNotificationSettingsRepository creates a new notification settings repository
func NewNotificationSettingsRepository(db *gorm.DB) *NotificationSettingsRepository {
	return &NotificationSettingsRepository{db: db}
}

// GetByUserID retrieves notification settings for a user
func (r *NotificationSettingsRepository) GetByUserID(userID string) (*models.NotificationSettings, error) {
	var settings models.NotificationSettings
	if err := r.db.Where("user_id = ?", userID).First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

// CreateOrUpdate creates or updates notification settings
func (r *NotificationSettingsRepository) CreateOrUpdate(settings *models.NotificationSettings) error {
	return r.db.Save(settings).Error
}

// Delete removes notification settings
func (r *NotificationSettingsRepository) Delete(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.NotificationSettings{}).Error
}

// GetAll retrieves all notification settings
func (r *NotificationSettingsRepository) GetAll() ([]models.NotificationSettings, error) {
	var settings []models.NotificationSettings
	if err := r.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}
