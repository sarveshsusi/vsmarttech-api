package service

import (
	"testing"
	"time"

	"rbac/models"
)

func TestAMCAssignmentCanClose(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		status string
		end    time.Time
		want   bool
	}{
		{
			name:   "active before end date",
			status: "active",
			end:    time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
			want:   false,
		},
		{
			name:   "active on end date",
			status: "active",
			end:    time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
			want:   false,
		},
		{
			name:   "active after end date",
			status: "active",
			end:    time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC),
			want:   true,
		},
		{
			name:   "expired status before end date",
			status: "expired",
			end:    time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
			want:   true,
		},
		{
			name:   "already closed",
			status: "closed",
			end:    time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AMCAssignmentCanClose(&models.AMCAssignment{
				Status:      tt.status,
				AMCEndDate:  tt.end,
			}, now)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
