package handler

import (
	"log"
	"net/http"
	"strconv"

	"rbac/models"
	"rbac/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notifService *service.NotificationService
}

func NewNotificationHandler(notifService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notifService: notifService,
	}
}

/* =========================
   GET NOTIFICATIONS
========================= */

func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		if num, err := strconv.Atoi(p); err == nil && num > 0 {
			page = num
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if num, err := strconv.Atoi(ps); err == nil && num > 0 && num <= 100 {
			pageSize = num
		}
	}

	notifications, total, err := h.notifService.GetUserNotifications(userID, page, pageSize)
	if err != nil {
		log.Printf("[GET_NOTIFICATIONS_ERROR] Failed to fetch notifications: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	// Ensure notifications is never null in JSON
	if notifications == nil {
		notifications = []models.Notification{}
	}

	log.Printf("[GET_NOTIFICATIONS_SUCCESS] user_id=%s count=%d total=%d", userID, len(notifications), total)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

/* =========================
   GET UNREAD COUNT
========================= */

func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	count, err := h.notifService.GetUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

/* =========================
   MARK AS READ
========================= */

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	notificationID := c.Param("id")
	id, err := uuid.Parse(notificationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.notifService.MarkAsRead(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

/* =========================
   MARK ALL AS READ
========================= */

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.notifService.MarkAllAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

/* =========================
   GET NOTIFICATION PREFERENCES
========================= */

func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	prefs, err := h.notifService.GetPreference(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch preferences"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

/* =========================
   UPDATE NOTIFICATION PREFERENCES
========================= */

type UpdatePreferencesRequest struct {
	EmailNotifications             *bool `json:"email_notifications,omitempty"`
	InAppNotifications             *bool `json:"in_app_notifications,omitempty"`
	WebhookNotifications           *bool `json:"webhook_notifications,omitempty"`
	TicketCreatedNotification      *bool `json:"ticket_created_notification,omitempty"`
	TicketAssignedNotification     *bool `json:"ticket_assigned_notification,omitempty"`
	TicketStatusChangeNotification *bool `json:"ticket_status_change_notification,omitempty"`
	TicketClosedNotification       *bool `json:"ticket_closed_notification,omitempty"`
}

func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req UpdatePreferencesRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	updates := make(map[string]interface{})
	if req.EmailNotifications != nil {
		updates["email_notifications"] = *req.EmailNotifications
	}
	if req.InAppNotifications != nil {
		updates["in_app_notifications"] = *req.InAppNotifications
	}
	if req.WebhookNotifications != nil {
		updates["webhook_notifications"] = *req.WebhookNotifications
	}
	if req.TicketCreatedNotification != nil {
		updates["ticket_created_notification"] = *req.TicketCreatedNotification
	}
	if req.TicketAssignedNotification != nil {
		updates["ticket_assigned_notification"] = *req.TicketAssignedNotification
	}
	if req.TicketStatusChangeNotification != nil {
		updates["ticket_status_change_notification"] = *req.TicketStatusChangeNotification
	}
	if req.TicketClosedNotification != nil {
		updates["ticket_closed_notification"] = *req.TicketClosedNotification
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if err := h.notifService.UpdatePreference(userID, updates); err != nil {
		log.Printf("[PREFERENCE_ERROR] Failed to update preferences: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}

/* =========================
   TEST: MANUALLY CREATE NOTIFICATION
========================= */

func (h *NotificationHandler) TestCreateNotification(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// For testing - create a simple notification
	testMsg := "This is a test notification"
	title := "Test Notification"

	if err := h.notifService.CreateTicketNotification(
		userID,
		"TEST/01/01/1",
		models.NotificationTypeTicketCreated,
		title,
		testMsg,
		nil,
		nil,
	); err != nil {
		log.Printf("[TEST_NOTIFICATION_ERROR] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test notification created",
		"user_id": userID,
	})
}
