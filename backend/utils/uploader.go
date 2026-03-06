package utils

import "mime/multipart"

type AuthTokenResponse struct {
	Token     string `json:"token"`
	Expire    string `json:"expire"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

type ImageUploader interface {
	// Upload file and return URL
	Upload(file *multipart.FileHeader) (string, error)

	// GenerateAuthToken is deprecated (was for ImageKit)
	GenerateAuthToken() (*AuthTokenResponse, error)
}
