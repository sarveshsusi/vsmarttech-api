package notify

import (
	"github.com/gin-gonic/gin"

	"rbac/handler"
)

// Register mounts notification endpoints for all authenticated roles.
func Register(protected *gin.RouterGroup, h *handler.NotificationHandler) {
	notifications := protected.Group("/notifications")
	{
		notifications.GET("", h.GetNotifications)
		notifications.GET("/unread", h.GetUnreadCount)
		notifications.PUT("/:id/read", h.MarkAsRead)
		notifications.PUT("/all/read", h.MarkAllAsRead)
		notifications.GET("/preferences", h.GetPreferences)
		notifications.PUT("/preferences", h.UpdatePreferences)
		notifications.POST("/test", h.TestCreateNotification)
	}
}
