package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"rbac/models"
)

type RememberedDeviceRepo struct {
	db *gorm.DB
}

func NewRememberedDeviceRepo(db *gorm.DB) *RememberedDeviceRepo {
	return &RememberedDeviceRepo{db}
}

func (r *RememberedDeviceRepo) Create(rd *models.RememberedDevice) error {
	return r.db.Create(rd).Error
}

func (r *RememberedDeviceRepo) ExistsValid(
	userID uuid.UUID,
	hashedToken string,
) bool {
	err := r.db.Where(
		"user_id = ? AND token = ? AND expires_at > NOW()",
		userID, hashedToken,
	).First(&models.RememberedDevice{}).Error

	return err == nil
}

func (r *RememberedDeviceRepo) DeleteByUser(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).
		Delete(&models.RememberedDevice{}).Error
}
