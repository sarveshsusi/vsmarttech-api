package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"log"
	"mime/multipart"
	"strconv"
	"time"

	"rbac/config"
)

type ImageKitUploader struct {
	publicKey  string
	privateKey string
	endpoint   string
}

func NewImageKitUploader(cfg *config.Config) ImageUploader {
	return &ImageKitUploader{
		publicKey:  cfg.ImageKit.PublicKey,
		privateKey: cfg.ImageKit.PrivateKey,
		endpoint:   cfg.ImageKit.Endpoint,
	}
}

// GenerateAuthToken creates a signed authentication token for client-side uploads
// This uses ImageKit's client-side authentication without needing a token
func (u *ImageKitUploader) GenerateAuthToken() (*AuthTokenResponse, error) {
	// Token expiration: 30 minutes from now
	expire := time.Now().Add(30 * time.Minute).Unix()
	expireStr := strconv.FormatInt(expire, 10)

	// Create signature: SHA1(privateKey + expireTimestamp)
	// The signature must be calculated with the timestamp as a STRING
	stringToSign := u.privateKey + expireStr
	h := sha1.New()
	h.Write([]byte(stringToSign))
	signature := hex.EncodeToString(h.Sum(nil))

	// DEBUG: Log what we're sending
	log.Printf("[IMAGEKIT_AUTH] expire=%s | stringToSign=%s | signature=%s | publicKey=%s",
		expireStr, stringToSign, signature, u.publicKey)

	return &AuthTokenResponse{
		Token:     expireStr, // Timestamp as string (used by ImageKit)
		Expire:    expireStr, // Timestamp as string (must match signature calculation)
		PublicKey: u.publicKey,
		Signature: signature,
	}, nil
}

// Upload is deprecated - use client-side upload instead
func (u *ImageKitUploader) Upload(file *multipart.FileHeader) (string, error) {
	return "", errors.New("server-side upload is deprecated; use GenerateAuthToken for client-side upload")
}
