package utils

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

/*
=====================
 Safe Error Response Handler
=====================
Prevents information disclosure by converting specific errors
to generic messages while logging real errors internally.
*/

// ErrorResponse handles errors safely without leaking internal details
// Use this for ALL error responses instead of c.JSON(..., gin.H{"error": err.Error()})
func ErrorResponse(c *gin.Context, err error, defaultMsg string) {
	// Log the real error internally (for debugging/monitoring)
	log.Printf("[ERROR] %s: %v", defaultMsg, err)

	// Determine HTTP status based on error type
	statusCode := http.StatusInternalServerError
	publicMsg := defaultMsg

	// Handle specific error types safely
	if errors.Is(err, gorm.ErrRecordNotFound) {
		statusCode = http.StatusNotFound
		publicMsg = "resource not found"
	} else if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "Invalid") {
		statusCode = http.StatusBadRequest
		publicMsg = "invalid request"
	} else if strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "not found or access denied") {
		statusCode = http.StatusForbidden
		publicMsg = "ticket not found or access denied"
	} else if strings.Contains(err.Error(), "not found") {
		statusCode = http.StatusNotFound
		publicMsg = "resource not found"
	} else if strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "forbidden") || strings.Contains(err.Error(), "Forbidden") {
		statusCode = http.StatusForbidden
		publicMsg = "access denied"
	} else if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "Unauthorized") {
		statusCode = http.StatusUnauthorized
		publicMsg = "authentication required"
	} else if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "constraint") {
		statusCode = http.StatusConflict
		publicMsg = "this resource already exists"
	}

	c.JSON(statusCode, gin.H{"error": publicMsg})
}

// ValidateBindJSON safely validates JSON binding
// Returns error if validation fails (response is already sent)
// Use like: if err := utils.ValidateBindJSON(c, &req); err != nil { return }
func ValidateBindJSON(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		log.Printf("[VALIDATION_ERROR] %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request format",
		})
		return err
	}
	return nil
}

// ErrorResponseWithStatus allows custom status codes
func ErrorResponseWithStatus(c *gin.Context, statusCode int, err error, publicMsg string) {
	log.Printf("[ERROR] (HTTP %d) %s: %v", statusCode, publicMsg, err)
	c.JSON(statusCode, gin.H{"error": publicMsg})
}

// IsForeignKeyViolation returns true when err represents a Postgres
// foreign-key constraint violation (SQLSTATE 23503) — i.e. an attempt to
// delete (or update) a row that is still referenced by another table.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	// Fallback for wrapped/driver variations that don't surface *pgconn.PgError
	msg := err.Error()
	return strings.Contains(msg, "violates foreign key constraint") ||
		strings.Contains(msg, "SQLSTATE 23503")
}

// DeleteConflictResponse sends a clear, actionable 409 response when a
// delete fails because dependent records still reference the row, falling
// back to a generic message for any other error.
func DeleteConflictResponse(c *gin.Context, err error, entityName string) {
	log.Printf("[ERROR] delete %s failed: %v", entityName, err)

	if IsForeignKeyViolation(err) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Cannot delete " + entityName + ": it is still referenced by other existing records (e.g. tickets, assignments, or contracts). Remove or reassign those first.",
		})
		return
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": entityName + " not found"})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}
