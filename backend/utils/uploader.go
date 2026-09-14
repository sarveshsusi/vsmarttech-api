package utils

import (
	"mime/multipart"
	"strings"

	"rbac/config"
)

type AuthTokenResponse struct {
	Token     string `json:"token"`
	Expire    string `json:"expire"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

type ImageUploader interface {
	// Upload file and return URL (validates size; prefer UploadValidated for new callers)
	Upload(file *multipart.FileHeader) (string, error)

	// UploadValidated uploads already-sniffed image bytes with a server-chosen content type/name
	UploadValidated(data []byte, contentType string) (string, error)

	// OpenStored reads bytes for a previously stored proof URL/key (S3 or local).
	OpenStored(storedURL string) (data []byte, contentType string, err error)

	// GenerateAuthToken is deprecated (was for ImageKit)
	GenerateAuthToken() (*AuthTokenResponse, error)
}

func NewImageUploader(cfg *config.Config) (ImageUploader, error) {
	if cfg != nil && strings.EqualFold(cfg.Storage.Type, "s3") {
		return NewS3Uploader(cfg)
	}
	localDir := "./uploads"
	baseURL := "http://localhost:8080/uploads"
	if cfg != nil {
		if cfg.Storage.LocalDir != "" {
			localDir = cfg.Storage.LocalDir
		}
		if cfg.Storage.BaseURL != "" {
			baseURL = cfg.Storage.BaseURL
		}
	}
	return NewLocalUploader(localDir, baseURL), nil
}
