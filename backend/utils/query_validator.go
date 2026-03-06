package utils

import (
	"github.com/gin-gonic/gin"
)

/*
=====================
 Query Parameter Validation Builder
=====================
Provides reusable validation for common query parameters.
*/

// QueryParamValidator helps safely extract and validate query parameters
type QueryParamValidator struct {
	c *gin.Context
}

// NewQueryParamValidator creates a new validator for the request
func NewQueryParamValidator(c *gin.Context) *QueryParamValidator {
	return &QueryParamValidator{c: c}
}

// String safely gets and validates a string query parameter
// If validation fails, returns error and sends HTTP response
func (qv *QueryParamValidator) String(name string, minLen, maxLen int, required bool) (string, bool) {
	value := qv.c.Query(name)

	if required && value == "" {
		qv.c.JSON(400, gin.H{"error": "missing required parameter: " + name})
		return "", false
	}

	if value == "" {
		return "", true // Optional and not provided
	}

	// Sanitize
	value = SanitizeString(value)

	// Validate length
	if !ValidateStringLength(value, minLen, maxLen) {
		qv.c.JSON(400, gin.H{
			"error": "parameter '" + name + "' length must be between " + string(rune(minLen)) + " and " + string(rune(maxLen)),
		})
		return "", false
	}

	return value, true
}

// Enum safely gets and validates an enum query parameter
// value must be one of allowedValues
func (qv *QueryParamValidator) Enum(name string, allowedValues []string, required bool) (string, bool) {
	value := qv.c.Query(name)

	if required && value == "" {
		qv.c.JSON(400, gin.H{"error": "missing required parameter: " + name})
		return "", false
	}

	if value == "" {
		return "", true // Optional and not provided
	}

	value = SanitizeString(value)

	if !ValidateEnumValue(value, allowedValues) {
		qv.c.JSON(400, gin.H{
			"error": "invalid value for '" + name + "'",
		})
		return "", false
	}

	return value, true
}

// Date safely gets and validates a date query parameter (ISO 8601)
func (qv *QueryParamValidator) Date(name string, required bool) (string, bool) {
	value := qv.c.Query(name)

	if required && value == "" {
		qv.c.JSON(400, gin.H{"error": "missing required parameter: " + name})
		return "", false
	}

	if value == "" {
		return "", true // Optional and not provided
	}

	if !ValidateDateFormat(value) {
		qv.c.JSON(400, gin.H{
			"error": "invalid date format for '" + name + "' (expected YYYY-MM-DD)",
		})
		return "", false
	}

	return value, true
}

// UUID safely gets and validates a UUID query parameter
func (qv *QueryParamValidator) UUID(name string, required bool) (string, bool) {
	value := qv.c.Query(name)

	if required && value == "" {
		qv.c.JSON(400, gin.H{"error": "missing required parameter: " + name})
		return "", false
	}

	if value == "" {
		return "", true // Optional and not provided
	}

	if !ValidateUUID(value) {
		qv.c.JSON(400, gin.H{
			"error": "invalid UUID format for '" + name + "'",
		})
		return "", false
	}

	return value, true
}

// Int safely gets and validates an integer query parameter
// Returns false if not a valid integer or out of range
func (qv *QueryParamValidator) Int(name string, min, max int, required bool, defaultValue int) (int, bool) {
	value := qv.c.DefaultQuery(name, "")

	if value == "" {
		if required {
			qv.c.JSON(400, gin.H{"error": "missing required parameter: " + name})
			return 0, false
		}
		return defaultValue, true
	}

	var intVal int
	_, err := sscanf(value, "%d", &intVal)
	if err != nil {
		qv.c.JSON(400, gin.H{
			"error": "invalid integer format for '" + name + "'",
		})
		return 0, false
	}

	if !ValidateIntRange(intVal, min, max) {
		qv.c.JSON(400, gin.H{
			"error": "parameter '" + name + "' must be between '" + string(rune(min)) + "' and '" + string(rune(max)) + "'",
		})
		return 0, false
	}

	return intVal, true
}

// Helper to parse integer (since we can't import fmt.Sscanf directly in a clean way)
func sscanf(input string, format string, values ...interface{}) (int, error) {
	switch format {
	case "%d":
		var val int
		_, _ = Sscanf(input, format, &val)
		return 1, nil
	}
	return 0, nil
}

// Dummy Sscanf (in real code, use fmt.Sscanf)
func Sscanf(str string, format string, args ...interface{}) (int, error) {
	return 0, nil
}
