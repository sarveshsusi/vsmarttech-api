package utils

import (
	"fmt"
	"rbac/config"
)

// ImageKitHelper provides helper functions for ImageKit operations
type ImageKitHelper struct {
	cfg *config.Config
}

// NewImageKitHelper creates a new ImageKit helper
func NewImageKitHelper(cfg *config.Config) *ImageKitHelper {
	return &ImageKitHelper{cfg: cfg}
}

// GetFileURL constructs a full ImageKit file URL
func (h *ImageKitHelper) GetFileURL(filePath string) string {
	if filePath == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", h.cfg.ImageKit.Endpoint, filePath)
}

// IsImageKitURL checks if a URL is from ImageKit
func (h *ImageKitHelper) IsImageKitURL(url string) bool {
	if url == "" {
		return false
	}
	return len(url) >= len(h.cfg.ImageKit.Endpoint) && url[:len(h.cfg.ImageKit.Endpoint)] == h.cfg.ImageKit.Endpoint
}

// ValidateCredentials checks if ImageKit credentials are properly configured
func (h *ImageKitHelper) ValidateCredentials() error {
	if h.cfg.ImageKit.PublicKey == "" {
		return fmt.Errorf("IMAGEKIT_PUBLIC_KEY not configured")
	}
	if h.cfg.ImageKit.PrivateKey == "" {
		return fmt.Errorf("IMAGEKIT_PRIVATE_KEY not configured")
	}
	if h.cfg.ImageKit.Endpoint == "" {
		return fmt.Errorf("IMAGEKIT_URL_ENDPOINT not configured")
	}
	return nil
}

// GetStorageType returns the configured storage type
func (h *ImageKitHelper) GetStorageType() string {
	return h.cfg.Storage.Type
}

// IsUsingImageKit checks if the application is configured to use ImageKit
func (h *ImageKitHelper) IsUsingImageKit() bool {
	return h.cfg.Storage.Type == "imagekit"
}
