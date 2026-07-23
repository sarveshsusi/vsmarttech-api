package utils

import "mime/multipart"

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

	// GenerateAuthToken is deprecated (was for ImageKit)
	GenerateAuthToken() (*AuthTokenResponse, error)
}
