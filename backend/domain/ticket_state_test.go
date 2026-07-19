package domain

import (
	"testing"

	"rbac/models"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from models.TicketStatus
		to   models.TicketStatus
		want bool
	}{
		{"open to assigned", models.StatusOpen, models.StatusAssigned, true},
		{"open to closed", models.StatusOpen, models.StatusClosed, true},
		{"open to in progress", models.StatusOpen, models.StatusInProgress, false},
		{"assigned to in progress", models.StatusAssigned, models.StatusInProgress, true},
		{"assigned to closed", models.StatusAssigned, models.StatusClosed, true},
		{"in progress to closed", models.StatusInProgress, models.StatusClosed, true},
		{"in progress to open", models.StatusInProgress, models.StatusOpen, false},
		{"closed is terminal", models.StatusClosed, models.StatusOpen, false},
		{"closed to assigned", models.StatusClosed, models.StatusAssigned, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
