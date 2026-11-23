package service

import (
	"errors"
	"strings"
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
				Status:     tt.status,
				AMCEndDate: tt.end,
			}, now)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAMCLifecycleFilterUsesTheCloseDateRule(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	rows := []models.AMCAssignment{
		{Status: "active", AMCEndDate: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), Notes: "open"},
		{Status: "active", AMCEndDate: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC), Notes: "ends-today"},
		{Status: "active", AMCEndDate: time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC), Notes: "past"},
		{Status: "expired", AMCEndDate: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), Notes: "marked"},
		{Status: "closed", AMCEndDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Notes: "closed"},
	}

	current, err := FilterAdminAMCLifecycle(rows, "current", now)
	if err != nil {
		t.Fatal(err)
	}
	if notes := amcNotes(current); notes != "open,ends-today,closed" {
		t.Fatalf("admin current = %s", notes)
	}
	expired, err := FilterAdminAMCLifecycle(rows, "expired", now)
	if err != nil {
		t.Fatal(err)
	}
	if notes := amcNotes(expired); notes != "past,marked" {
		t.Fatalf("admin expired = %s", notes)
	}
	all, err := FilterAdminAMCLifecycle(rows, "", now)
	if err != nil || len(all) != len(rows) {
		t.Fatalf("omitted lifecycle = %d, err %v", len(all), err)
	}
	if _, err := FilterAdminAMCLifecycle(rows, "other", now); !errors.Is(err, ErrAMCLifecycle) {
		t.Fatalf("invalid lifecycle err = %v", err)
	}

	supportCurrent, err := FilterSupportAMCLifecycle(rows, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if notes := amcNotes(supportCurrent); notes != "open,ends-today" {
		t.Fatalf("support current = %s", notes)
	}
	supportExpired, err := FilterSupportAMCLifecycle(rows, "expired", now)
	if err != nil {
		t.Fatal(err)
	}
	if notes := amcNotes(supportExpired); notes != "past,marked" {
		t.Fatalf("support expired = %s", notes)
	}
}

func amcNotes(rows []models.AMCAssignment) string {
	notes := make([]string, len(rows))
	for i := range rows {
		notes[i] = rows[i].Notes
	}
	return strings.Join(notes, ",")
}
