package utils

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
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

// OpenStored reads a file previously written by this uploader.
func (u *LocalUploader) OpenStored(storedURL string) ([]byte, string, error) {
	name := filepath.Base(strings.ReplaceAll(storedURL, "\\", "/"))
	if name == "" || name == "." || name == ".." || strings.Contains(name, "..") {
		return nil, "", errors.New("invalid stored path")
	}
	path := filepath.Join(u.uploadDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	contentType := "application/octet-stream"
	if detected, err := DetectImageContentType(data); err == nil {
		contentType = detected
	}
	return data, contentType, nil
}

// GenerateAuthToken is not applicable for local uploads
// Only ImageKit uses signed tokens for client-side uploads
func (u *LocalUploader) GenerateAuthToken() (*AuthTokenResponse, error) {
	return nil, errors.New("auth tokens only supported for ImageKit storage")
}
