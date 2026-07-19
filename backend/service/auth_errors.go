package service

import "errors"

// Sentinel / typed auth errors (prefer these over string prefix matching).
var (
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrPasswordResetRequired  = errors.New("PASSWORD_RESET_REQUIRED")
	ErrAccountDisabled        = errors.New("account disabled")
)

// ErrTwoFARequired wraps a temporary 2FA token that the handler must return.
type ErrTwoFARequired struct {
	TempToken string
}

func (e *ErrTwoFARequired) Error() string {
	return "TWO_FA_REQUIRED"
}

func NewErrTwoFARequired(tempToken string) error {
	return &ErrTwoFARequired{TempToken: tempToken}
}
