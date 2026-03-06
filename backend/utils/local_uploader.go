package utils

import (
	"errors"
	"fmt"
	"io"
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
	// ✅ size check FIRST
	if file.Size > 5*1024*1024 {
		return "", errors.New("file too large (max 5MB)")
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// ✅ Generate unique filename
	fileName := fmt.Sprintf(
		"%d%s",
		time.Now().UnixNano(),
		filepath.Ext(file.Filename),
	)

	// ✅ Create file path
	filePath := filepath.Join(u.uploadDir, fileName)

	// ✅ Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// ✅ Copy file contents
	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(filePath)
		return "", err
	}

	// ✅ Return relative URL
	fileURL := u.baseURL + "/" + fileName
	return fileURL, nil
}

// GenerateAuthToken is not applicable for local uploads
// Only ImageKit uses signed tokens for client-side uploads
func (u *LocalUploader) GenerateAuthToken() (*AuthTokenResponse, error) {
	return nil, errors.New("auth tokens only supported for ImageKit storage")
}
