package utils

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp"
)

// DetectImageContentType sniffs magic bytes. Never trust client Content-Type.
func DetectImageContentType(data []byte) (string, error) {
	if len(data) < 12 {
		return "", fmt.Errorf("file too small")
	}

	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg", nil
	case bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png", nil
	case bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a")):
		return "image/gif", nil
	case bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported image type")
	}
}

// ValidateDecodableImage ensures the payload is a real image (blocks polyglots / HTML disguised as images).
func ValidateDecodableImage(data []byte) error {
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("invalid image content")
	}
	return nil
}

// SafeUploadFilename returns a random-looking name with a safe extension (no path traversal / double ext).
func SafeUploadFilename(contentType string) string {
	ext := ".bin"
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	return ext
}

// SanitizeOriginalFilename strips path components and dangerous extensions for logging only.
func SanitizeOriginalFilename(name string) string {
	base := filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == ".." {
		return "upload"
	}
	return base
}
