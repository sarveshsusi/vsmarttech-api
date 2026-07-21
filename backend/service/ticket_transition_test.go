package service

import (
	"errors"
	"testing"

	"rbac/models"
)

func TestAssertTicketTransition(t *testing.T) {
	if err := assertTicketTransition(models.StatusOpen, models.StatusAssigned); err != nil {
		t.Fatalf("open->assigned: %v", err)
	}
	if err := assertTicketTransition(models.StatusOpen, models.StatusInProgress); err == nil {
		t.Fatal("open->in progress should fail")
	}
	if err := assertTicketTransition(models.StatusClosed, models.StatusAssigned); err != nil {
		t.Fatalf("closed->assigned: %v", err)
	}
	if err := assertTicketTransition(models.StatusClosed, models.StatusOpen); err != nil {
		t.Fatalf("closed->open: %v", err)
	}
	if err := assertTicketTransition(models.StatusClosed, models.StatusInProgress); err == nil {
		t.Fatal("closed->in progress should fail")
	}
}

func TestErrTwoFARequired(t *testing.T) {
	err := NewErrTwoFARequired("temp-token")
	var twoFA *ErrTwoFARequired
	if !errors.As(err, &twoFA) {
		t.Fatal("expected ErrTwoFARequired")
	}
	if twoFA.TempToken != "temp-token" {
		t.Fatalf("token=%q", twoFA.TempToken)
	}
	if !errors.Is(ErrPasswordResetRequired, ErrPasswordResetRequired) {
		t.Fatal("sentinel mismatch")
	}
}
