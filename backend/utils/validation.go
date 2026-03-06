package utils

import (
	"regexp"
	"strings"
)

/*
=====================
 Input Validation & Sanitization
=====================
Provides safe methods to validate and sanitize user inputs
to prevent injection attacks and malformed data.
*/

// SanitizeString removes dangerous characters and trims whitespace
// ✅ Removes NULL bytes
// ✅ Removes control characters
// ✅ Trims leading/trailing whitespace
func SanitizeString(s string) string {
	// Remove NULL bytes (can break C string comparisons in some contexts)
	s = strings.ReplaceAll(s, "\x00", "")
	// Remove other control characters (0x00-0x1F except tab/newline)
	for i := 0; i <= 0x1F; i++ {
		if i != '\t' && i != '\n' && i != '\r' {
			s = strings.ReplaceAll(s, string(rune(i)), "")
		}
	}
	// Trim whitespace
	s = strings.TrimSpace(s)
	return s
}

// ValidateEnumValue checks if input is in allowed set (case-sensitive)
// Example: ValidateEnumValue("open", []string{"open", "closed", "pending"})
func ValidateEnumValue(value string, allowed []string) bool {
	for _, v := range allowed {
		if value == v {
			return true
		}
	}
	return false
}

// ValidateEnumValueCI checks if input is in allowed set (case-insensitive)
func ValidateEnumValueCI(value string, allowed []string) bool {
	valueLower := strings.ToLower(value)
	for _, v := range allowed {
		if valueLower == strings.ToLower(v) {
			return true
		}
	}
	return false
}

// ValidateUUID checks if string is valid UUID format (v4)
// Accepts both with and without dashes
func ValidateUUID(s string) bool {
	re := regexp.MustCompile(`^[0-9a-f]{8}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{4}-?[0-9a-f]{12}$`)
	return re.MatchString(strings.ToLower(s))
}

// ValidateDateFormat checks if string matches ISO 8601 date format (YYYY-MM-DD)
func ValidateDateFormat(dateStr string) bool {
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	return re.MatchString(dateStr)
}

// ValidateEmail checks if string is valid email format
func ValidateEmail(email string) bool {
	// RFC 5322 simplified (not comprehensive but prevents obvious injection)
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email) && len(email) < 254
}

// ValidateStringLength checks if string length is within bounds
func ValidateStringLength(s string, minLen, maxLen int) bool {
	return len(s) >= minLen && len(s) <= maxLen
}

// ValidateIntRange checks if integer is within bounds
func ValidateIntRange(value, min, max int) bool {
	return value >= min && value <= max
}

// TruncateString safely truncates string to max length
func TruncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// ValidateAlphanumeric checks if string contains only alphanumeric characters and allowed symbols
func ValidateAlphanumeric(s string, allowedSymbols string) bool {
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			strings.ContainsRune(allowedSymbols, ch)) {
			return false
		}
	}
	return true
}

// ValidateSQLIdentifier validates database identifiers (table names, column names)
// ⚠️ Only for identifiers, NOT values (use parameterized queries for values!)
// Allows: [a-zA-Z0-9_]
func ValidateSQLIdentifier(identifier string) bool {
	re := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return re.MatchString(identifier) && len(identifier) <= 63 // PostgreSQL limit
}
