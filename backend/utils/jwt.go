package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"rbac/models"
)

/*
=====================
 JWT Claims
=====================
*/

// IMPORTANT:
// - UUID is stored as STRING (never binary)
// - `sub` is the source of truth
type Claims struct {
	UserID string      `json:"user_id"` // UUID as string
	Email  string      `json:"email"`
	Role   models.Role `json:"role"`
	jwt.RegisteredClaims
}

/*
=====================
 Access Token
=====================
*/

func GenerateAccessToken(
	user *models.User,
	secret string,
	expiry time.Duration,
) (string, error) {

	now := time.Now()

	claims := Claims{
		UserID: user.ID.String(), // ✅ UUID → string
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "ticketing-api",
			Subject:   user.ID.String(), // ✅ UUID in `sub`
			Audience:  []string{"ticketing-web"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			ID:        uuid.NewString(), // jti
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

/*
=====================
 Refresh Token
=====================
*/

func GenerateRefreshToken(
	secret string,
	expiry time.Duration,
) (string, error) {

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    "ticketing-api",
		Audience:  []string{"ticketing-web"},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		ID:        uuid.NewString(), // jti (used for rotation detection)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

/*
=====================
 Validate Token
=====================
*/

func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	// Extra safety: ensure UUID is valid
	if _, err := uuid.Parse(claims.UserID); err != nil {
		return nil, errors.New("invalid user_id in token")
	}

	return claims, nil
}
