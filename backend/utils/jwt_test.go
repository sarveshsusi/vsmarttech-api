package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"rbac/models"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	secret := "test-access-secret"
	user := &models.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Role:  models.RoleAdmin,
	}

	token, err := GenerateAccessToken(user, secret, time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != user.ID.String() {
		t.Fatalf("UserID = %q, want %q", claims.UserID, user.ID.String())
	}
	if claims.Email != user.Email {
		t.Fatalf("Email = %q, want %q", claims.Email, user.Email)
	}
	if claims.Role != models.RoleAdmin {
		t.Fatalf("Role = %q, want %q", claims.Role, models.RoleAdmin)
	}
}

func TestValidateTokenRejectsWrongSecret(t *testing.T) {
	user := &models.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Role:  models.RoleSupport,
	}
	token, err := GenerateAccessToken(user, "correct-secret", time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	if _, err := ValidateToken(token, "wrong-secret"); err == nil {
		t.Fatal("expected validation error with wrong secret")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken("refresh-secret", 24*time.Hour)
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty refresh token")
	}
}
