package service

import (
	"time"

	"rbac/models"
)

// ResolveSLAHours is the single source of truth for ticket SLA duration.
//
//	Priority   | Default | AMC service call
//	-----------|---------|------------------
//	Critical   | 24h     | 24h
//	Standard   | 72h     | 24h
//	Low        | 120h    | 48h
func ResolveSLAHours(priority models.TicketPriority, serviceCallType models.ServiceCallType) int {
	isAMC := serviceCallType == models.ServiceTypeAMC

	switch priority {
	case models.PriorityCritical:
		return 24
	case models.PriorityLow:
		if isAMC {
			return 48
		}
		return 120
	case models.PriorityStandard:
		fallthrough
	default:
		if isAMC {
			return 24
		}
		return 72
	}
}

// ComputeSLATarget returns sla hours and absolute deadline from a clock-start time.
func ComputeSLATarget(
	priority models.TicketPriority,
	serviceCallType models.ServiceCallType,
	clockStart time.Time,
) (slaHours int, targetAt time.Time) {
	slaHours = ResolveSLAHours(priority, serviceCallType)
	targetAt = clockStart.Add(time.Duration(slaHours) * time.Hour)
	return slaHours, targetAt
}

// EffectiveSLATarget prefers the stored deadline; falls back to recomputing
// from sla_hours or the canonical policy when older rows are incomplete.
func EffectiveSLATarget(ticket *models.Ticket, now time.Time) (targetAt time.Time, slaHours int) {
	if ticket == nil {
		return now, 72
	}

	slaHours = ticket.SLAHours
	if slaHours <= 0 {
		slaHours = ResolveSLAHours(ticket.Priority, ticket.ServiceCallType)
	}

	if ticket.TargetAt != nil && !ticket.TargetAt.IsZero() {
		return *ticket.TargetAt, slaHours
	}

	clockStart := ticket.CreatedAt
	if clockStart.IsZero() {
		clockStart = now
	}
	return clockStart.Add(time.Duration(slaHours) * time.Hour), slaHours
}
