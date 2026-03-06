package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db}
}

/* =========================
   CREATE NOTIFICATION
========================= */

func (r *NotificationRepository) Create(notification *models.Notification) error {
	return r.db.Create(notification).Error
}

/* =========================
   GET NOTIFICATIONS FOR USER
========================= */

func (r *NotificationRepository) GetUserNotifications(userID uuid.UUID, limit int, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

/* =========================
   GET UNREAD NOTIFICATIONS COUNT
========================= */

func (r *NotificationRepository) GetUnreadCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.
		Where("user_id = ? AND is_read = ?", userID, false).
		Model(&models.Notification{}).
		Count(&count).Error
	return count, err
}

/* =========================
   MARK AS READ
========================= */

func (r *NotificationRepository) MarkAsRead(notificationID uuid.UUID) error {
	now := gorm.Expr("CURRENT_TIMESTAMP")
	return r.db.
		Model(&models.Notification{}).
		Where("id = ?", notificationID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

/* =========================
   MARK ALL AS READ
========================= */

func (r *NotificationRepository) MarkAllAsRead(userID uuid.UUID) error {
	now := gorm.Expr("CURRENT_TIMESTAMP")
	return r.db.
		Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

/* =========================
   DELETE NOTIFICATION
========================= */

func (r *NotificationRepository) Delete(notificationID uuid.UUID) error {
	return r.db.Delete(&models.Notification{}, "id = ?", notificationID).Error
}

/* =========================
   CREATE WEBHOOK EVENT
========================= */

func (r *NotificationRepository) CreateWebhookEvent(event *models.WebhookEvent) error {
	return r.db.Create(event).Error
}

/* =========================
   GET UNDELIVERED WEBHOOK EVENTS
========================= */

func (r *NotificationRepository) GetUndeliveredWebhookEvents(limit int) ([]models.WebhookEvent, error) {
	var events []models.WebhookEvent
	err := r.db.
		Where("is_delivered = ? AND retry_count < ?", false, 5).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

/* =========================
   UPDATE WEBHOOK EVENT
========================= */

func (r *NotificationRepository) UpdateWebhookEvent(eventID uuid.UUID, updates map[string]interface{}) error {
	return r.db.
		Model(&models.WebhookEvent{}).
		Where("id = ?", eventID).
		Updates(updates).Error
}

/* =========================
   GET NOTIFICATION PREFERENCE
========================= */

func (r *NotificationRepository) GetPreference(userID uuid.UUID) (*models.NotificationPreference, error) {
	var pref models.NotificationPreference
	err := r.db.Where("user_id = ?", userID).First(&pref).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default preferences
			pref = models.NotificationPreference{
				ID:                             uuid.New(),
				UserID:                         userID,
				EmailNotifications:             true,
				InAppNotifications:             true,
				WebhookNotifications:           true,
				TicketCreatedNotification:      true,
				TicketAssignedNotification:     true,
				TicketStatusChangeNotification: true,
				TicketClosedNotification:       true,
			}
			if err := r.db.Create(&pref).Error; err != nil {
				return nil, err
			}
			return &pref, nil
		}
		return nil, err
	}
	return &pref, nil
}

/* =========================
   UPDATE NOTIFICATION PREFERENCE
========================= */

func (r *NotificationRepository) UpdatePreference(userID uuid.UUID, updates map[string]interface{}) error {
	return r.db.
		Model(&models.NotificationPreference{}).
		Where("user_id = ?", userID).
		Updates(updates).Error
}

/* =========================
   CHECK IF CONTRACT NOTIFICATION ALREADY SENT
========================= */

func (r *NotificationRepository) HasContractNotificationBeenSent(
	customerSolutionID uuid.UUID,
	notificationType models.NotificationType,
) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.Notification{}).
		Where("customer_solution_id = ? AND type = ?", customerSolutionID, notificationType).
		Count(&count).Error
	return count > 0, err
}
