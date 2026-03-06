package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SecurityAuditEvent represents a single security audit log entry
type SecurityAuditEvent struct {
	RequestID    string
	Timestamp    time.Time
	ClientIP     string
	Method       string
	Path         string
	StatusCode   int
	ResponseTime time.Duration
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	Changes      map[string]interface{}
	ErrorMessage string
	Success      bool
}

// SecurityAuditMiddleware provides enhanced security audit logging with detailed event tracking
// Does NOT change business logic, just records events for security monitoring
func SecurityAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate unique request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Start timer
		startTime := time.Now()

		// Get user ID if authenticated
		userID := ""
		if user, exists := c.Get("user_id"); exists {
			userID = fmt.Sprintf("%v", user)
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Capture request details
		method := c.Request.Method
		path := c.Request.URL.Path

		// Log request received
		logSecurityAuditEvent(&SecurityAuditEvent{
			RequestID: requestID,
			Timestamp: startTime,
			ClientIP:  clientIP,
			Method:    method,
			Path:      path,
			UserID:    userID,
			Action:    "REQUEST_RECEIVED",
		})

		// Process request
		c.Next()

		// Calculate response time
		responseTime := time.Since(startTime)

		// Get response status
		statusCode := c.Writer.Status()

		// Log request completed
		logSecurityAuditEvent(&SecurityAuditEvent{
			RequestID:    requestID,
			Timestamp:    time.Now(),
			ClientIP:     clientIP,
			Method:       method,
			Path:         path,
			StatusCode:   statusCode,
			ResponseTime: responseTime,
			UserID:       userID,
			Action:       "REQUEST_COMPLETED",
			Success:      statusCode >= 200 && statusCode < 300,
		})

		// Log specific actions based on method
		switch method {
		case "POST":
			logSecurityAuditEvent(&SecurityAuditEvent{
				RequestID:  requestID,
				Timestamp:  time.Now(),
				ClientIP:   clientIP,
				Method:     method,
				Path:       path,
				UserID:     userID,
				Action:     "CREATE",
				StatusCode: statusCode,
				Success:    statusCode == 201 || statusCode == 200,
			})
		case "PUT", "PATCH":
			logSecurityAuditEvent(&SecurityAuditEvent{
				RequestID:  requestID,
				Timestamp:  time.Now(),
				ClientIP:   clientIP,
				Method:     method,
				Path:       path,
				UserID:     userID,
				Action:     "UPDATE",
				StatusCode: statusCode,
				Success:    statusCode == 200,
			})
		case "DELETE":
			logSecurityAuditEvent(&SecurityAuditEvent{
				RequestID:  requestID,
				Timestamp:  time.Now(),
				ClientIP:   clientIP,
				Method:     method,
				Path:       path,
				UserID:     userID,
				Action:     "DELETE",
				StatusCode: statusCode,
				Success:    statusCode == 200 || statusCode == 204,
			})
		}

		// Log authentication events
		if path == "/api/v1/auth/login" && statusCode == 200 {
			logSecurityAuditEvent(&SecurityAuditEvent{
				RequestID:  requestID,
				Timestamp:  time.Now(),
				ClientIP:   clientIP,
				UserID:     userID,
				Action:     "LOGIN_SUCCESS",
				StatusCode: statusCode,
				Success:    true,
			})
		} else if path == "/api/v1/auth/login" && statusCode >= 400 {
			logSecurityAuditEvent(&SecurityAuditEvent{
				RequestID:    requestID,
				Timestamp:    time.Now(),
				ClientIP:     clientIP,
				Action:       "LOGIN_FAILED",
				StatusCode:   statusCode,
				Success:      false,
				ErrorMessage: "Invalid credentials or account locked",
			})
		}

		// Log suspicious activity
		if statusCode == 403 {
			logSecurityAuditEvent(&SecurityAuditEvent{
				RequestID:  requestID,
				Timestamp:  time.Now(),
				ClientIP:   clientIP,
				UserID:     userID,
				Path:       path,
				Action:     "ACCESS_DENIED",
				StatusCode: statusCode,
				Success:    false,
			})
		}

		// Log rate limit violations
		if statusCode == 429 {
			logSecurityAuditEvent(&SecurityAuditEvent{
				RequestID:  requestID,
				Timestamp:  time.Now(),
				ClientIP:   clientIP,
				UserID:     userID,
				Action:     "RATE_LIMIT_EXCEEDED",
				StatusCode: statusCode,
				Success:    false,
			})
		}
	}
}

// logSecurityAuditEvent logs a security audit event to file and console
func logSecurityAuditEvent(auditLog *SecurityAuditEvent) {
	timestamp := auditLog.Timestamp.Format("2006-01-02 15:04:05")

	logMessage := fmt.Sprintf(
		"[SECURITY_AUDIT] %s | RequestID=%s | Action=%s | User=%s | IP=%s | Method=%s | Path=%s | Status=%d | Time=%v | Success=%v | Error=%s",
		timestamp,
		auditLog.RequestID,
		auditLog.Action,
		auditLog.UserID,
		auditLog.ClientIP,
		auditLog.Method,
		auditLog.Path,
		auditLog.StatusCode,
		auditLog.ResponseTime,
		auditLog.Success,
		auditLog.ErrorMessage,
	)

	// Log to console and file
	log.Println(logMessage)

	// TODO: Log to audit trail database
	// TODO: Send to security monitoring service
}

// LogSecurityAuditEvent logs a custom security-related event
// Can be called from handlers to log important security decisions
func LogSecurityAuditEvent(requestID, userID, action, details string) {
	logSecurityAuditEvent(&SecurityAuditEvent{
		RequestID:    requestID,
		Timestamp:    time.Now(),
		UserID:       userID,
		Action:       action,
		ErrorMessage: details,
		Success:      true,
	})
}

// LogSecurityAuditViolation logs a potential security violation
func LogSecurityAuditViolation(requestID, userID, clientIP, action, details string) {
	logSecurityAuditEvent(&SecurityAuditEvent{
		RequestID:    requestID,
		Timestamp:    time.Now(),
		ClientIP:     clientIP,
		UserID:       userID,
		Action:       action,
		ErrorMessage: details,
		Success:      false,
	})
}
