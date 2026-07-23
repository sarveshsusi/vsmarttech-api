package utils

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

type LocalUploader struct {
	uploadDir string
	baseURL   string
}

func NewLocalUploader(uploadDir string, baseURL string) ImageUploader {
	// Create upload directory if it doesn't exist
	os.MkdirAll(uploadDir, 0755)
	return &LocalUploader{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}
}

func (u *LocalUploader) Upload(file *multipart.FileHeader) (string, error) {
	if file.Size > 1*1024*1024 {
		return "", errors.New("file too large (max 1MB)")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	fileBytes := make([]byte, file.Size)
	if _, err := src.Read(fileBytes); err != nil {
		return "", err
	}

	contentType, err := DetectImageContentType(fileBytes)
	if err != nil {
		return "", errors.New("unsupported image type")
	}
	if err := ValidateDecodableImage(fileBytes); err != nil {
		return "", err
	}

	return u.UploadValidated(fileBytes, contentType)
}

func (u *LocalUploader) UploadValidated(data []byte, contentType string) (string, error) {
	if len(data) > 1*1024*1024 {
		return "", errors.New("file too large (max 1MB)")
	}

	ext := SafeUploadFilename(contentType)
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(u.uploadDir, fileName)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	return u.baseURL + "/" + fileName, nil
}

// GenerateAuthToken is not applicable for local uploads
// Only ImageKit uses signed tokens for client-side uploads
func (u *LocalUploader) GenerateAuthToken() (*AuthTokenResponse, error) {
	return nil, errors.New("auth tokens only supported for ImageKit storage")
}
