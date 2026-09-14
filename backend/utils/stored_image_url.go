package utils

import (
	"net/url"
	"strings"
)

// IsAllowedStoredImageURL accepts empty (clear) or http(s) URLs from our
// upload storage (S3, local /uploads, localhost).
func IsAllowedStoredImageURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}

	lower := strings.ToLower(raw)
	if strings.Contains(lower, "javascript:") || strings.Contains(lower, "data:") {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return false
	}

	host := strings.ToLower(u.Host)
	path := strings.ToLower(u.Path)
	return strings.Contains(host, "s3") ||
		strings.Contains(host, "localhost") ||
		strings.Contains(host, "127.0.0.1") ||
		strings.Contains(path, "/uploads/")
}
