package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TwoFATokenClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Remember bool      `json:"remember"`
	jwt.RegisteredClaims
}

func Generate2FAToken(
	userID uuid.UUID,
	remember bool,
	secret string,
) (string, error) {

	claims := TwoFATokenClaims{
		UserID:   userID,
		Remember: remember,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	).SignedString([]byte(secret))
}

func Parse2FAToken(raw, secret string) (*TwoFATokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		raw,
		&TwoFATokenClaims{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*TwoFATokenClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
