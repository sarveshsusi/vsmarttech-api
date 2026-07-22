package service

import (
	"testing"
	"time"

	"rbac/models"
)

func TestResolveSLAHours(t *testing.T) {
	cases := []struct {
		priority models.TicketPriority
		svc      models.ServiceCallType
		want     int
	}{
		{models.PriorityCritical, "", 24},
		{models.PriorityStandard, "", 72},
		{models.PriorityLow, "", 120},
		{models.PriorityStandard, models.ServiceTypeAMC, 24},
		{models.PriorityLow, models.ServiceTypeAMC, 48},
		{models.PriorityCritical, models.ServiceTypeAMC, 24},
	}
	for _, tc := range cases {
		got := ResolveSLAHours(tc.priority, tc.svc)
		if got != tc.want {
			t.Fatalf("ResolveSLAHours(%s,%s)=%d want %d", tc.priority, tc.svc, got, tc.want)
		}
	}
}

func TestEffectiveSLATargetUsesStoredDeadline(t *testing.T) {
	deadline := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	ticket := &models.Ticket{
		Priority: models.PriorityStandard,
		SLAHours: 72,
		TargetAt: &deadline,
	}
	got, hours := EffectiveSLATarget(ticket, time.Now().UTC())
	if !got.Equal(deadline) {
		t.Fatalf("expected stored target, got %v", got)
	}
	if hours != 72 {
		t.Fatalf("hours=%d", hours)
	}
}
