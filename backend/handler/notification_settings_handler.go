package handler

import (
	"net/http"
	"rbac/models"
	"rbac/repository"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotificationSettingsHandler handles notification settings requests
type NotificationSettingsHandler struct {
	repo *repository.NotificationSettingsRepository
}

// NewNotificationSettingsHandler creates a new notification settings handler
func NewNotificationSettingsHandler(db *gorm.DB) *NotificationSettingsHandler {
	return &NotificationSettingsHandler{
		repo: repository.NewNotificationSettingsRepository(db),
	}
}

// GetSettings retrieves notification settings for a user
func (h *NotificationSettingsHandler) GetSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	settings, err := h.repo.GetByUserID(userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates notification settings for a user
func (h *NotificationSettingsHandler) UpdateSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.NotificationSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.UserID = userID

	if err := h.repo.CreateOrUpdate(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings updated successfully"})
}

// DeleteSettings deletes notification settings for a user
func (h *NotificationSettingsHandler) DeleteSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.repo.Delete(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings deleted successfully"})
}
