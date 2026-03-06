package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate unique request ID for tracking
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Get user ID if authenticated
		userID := ""
		if user, exists := c.Get("user_id"); exists {
			userID = fmt.Sprintf("%v", user)
		}

		c.Next()

		// Enhanced audit logging with more details
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()

		// Log to file
		log.Printf(
			"[AUDIT] RequestID=%s | Method=%s | Path=%s | Status=%d | IP=%s | User=%s | Duration=%v",
			requestID,
			method,
			path,
			statusCode,
			ip,
			userID,
			duration,
		)

		// Log specific security events
		switch statusCode {
		case 401:
			log.Printf("[SECURITY] Unauthorized access attempt - RequestID=%s | IP=%s | Path=%s", requestID, ip, path)
		case 403:
			log.Printf("[SECURITY] Access denied (forbidden) - RequestID=%s | User=%s | IP=%s | Path=%s", requestID, userID, ip, path)
		case 429:
			log.Printf("[SECURITY] Rate limit exceeded - RequestID=%s | IP=%s | Endpoint=%s", requestID, ip, path)
		case 400:
			log.Printf("[SECURITY] Bad request - RequestID=%s | IP=%s | Path=%s", requestID, ip, path)
		}

		// Log successful authentication
		if path == "/api/v1/auth/login" && statusCode == 200 {
			log.Printf("[AUTH] Successful login - RequestID=%s | User=%s | IP=%s | Time=%v", requestID, userID, ip, duration)
		}

		// Log authentication failures
		if path == "/api/v1/auth/login" && statusCode >= 400 {
			log.Printf("[AUTH_FAIL] Login failed - RequestID=%s | IP=%s | Status=%d", requestID, ip, statusCode)
		}

		// Log data modifications
		if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
			action := "UNKNOWN"
			switch method {
			case "POST":
				action = "CREATE"
			case "PUT", "PATCH":
				action = "UPDATE"
			case "DELETE":
				action = "DELETE"
			}
			if statusCode >= 200 && statusCode < 300 {
				log.Printf("[DATA_CHANGE] %s - RequestID=%s | User=%s | Path=%s | Status=%d", action, requestID, userID, path, statusCode)
			}
		}
	}
}
