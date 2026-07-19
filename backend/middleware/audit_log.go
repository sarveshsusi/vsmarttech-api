package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/database"
	"rbac/models"
)

// AuditLog records request metadata to slog and persists a row to audit_logs when DB is ready.
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID, _ := c.Get(CtxRequestID)
		if requestID == nil || requestID == "" {
			requestID = uuid.NewString()
			c.Set(CtxRequestID, requestID)
		}

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()

		userID := uuid.Nil
		if raw, exists := c.Get(CtxUserID); exists {
			if id, ok := raw.(uuid.UUID); ok {
				userID = id
			}
		}

		slog.Info("audit",
			"request_id", fmt.Sprint(requestID),
			"method", method,
			"path", path,
			"status", statusCode,
			"ip", ip,
			"user_id", userID,
			"duration_ms", duration.Milliseconds(),
		)

		if database.DB == nil {
			return
		}

		action := method + " " + path
		_ = database.DB.Create(&models.AuditLog{
			Entity:      "http_request",
			EntityID:    uuid.Nil,
			Action:      truncate(action, 250),
			PerformedBy: userID,
			IP:          ip,
			UserAgent:   truncate(c.Request.UserAgent(), 250),
		}).Error
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
