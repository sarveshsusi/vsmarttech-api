package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CtxRequestID = "request_id"

// RequestIDMiddleware assigns an X-Request-ID to every request and Gin context.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(CtxRequestID, rid)
		c.Writer.Header().Set("X-Request-ID", rid)
		c.Next()
	}
}

// StructuredAccessLog logs method, path, status, latency, and request ID via slog.
func StructuredAccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		rid, _ := c.Get(CtxRequestID)
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"request_id", rid,
		}
		if uid, ok := c.Get(CtxUserID); ok {
			attrs = append(attrs, "user_id", uid)
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			slog.Error("http_request", attrs...)
		case status >= 400:
			slog.Warn("http_request", attrs...)
		default:
			slog.Info("http_request", attrs...)
		}
	}
}
