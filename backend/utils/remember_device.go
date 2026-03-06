package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func HashRememberDeviceToken(token string) string {
	secret := os.Getenv("REMEMBER_DEVICE_SECRET")
	if secret == "" {
		panic("REMEMBER_DEVICE_SECRET not set")
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}
