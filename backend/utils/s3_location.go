package utils

import (
	"net/url"
	"strings"
)

// ParseS3ObjectRef extracts bucket and object key from an S3 URL or raw key.
func ParseS3ObjectRef(raw, fallbackBucket string) (bucket, key string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", "", false
	}

	if u.Scheme == "" || u.Host == "" {
		if fallbackBucket == "" {
			return "", "", false
		}
		key = strings.TrimPrefix(strings.ReplaceAll(raw, "\\", "/"), "/")
		if key == "" || strings.Contains(key, "..") {
			return "", "", false
		}
		return fallbackBucket, key, true
	}

	host := u.Host
	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return "", "", false
	}

	if i := strings.Index(host, ".s3."); i > 0 {
		return host[:i], path, true
	}
	if strings.HasPrefix(host, "s3.") || host == "s3.amazonaws.com" {
		bucket, rest, found := strings.Cut(path, "/")
		if !found || bucket == "" || rest == "" {
			return "", "", false
		}
		return bucket, rest, true
	}

	return "", "", false
}
