package models

import "time"

// NotificationSettings represents user notification preferences
type NotificationSettings struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	UserID        string    `gorm:"index" json:"user_id"`
	EmailNotified bool      `json:"email_notified"`
	SMSNotified   bool      `json:"sms_notified"`
	InAppNotified bool      `json:"in_app_notified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName specifies the table name for NotificationSettings
func (NotificationSettings) TableName() string {
	return "notification_settings"
}
