package service

import (
	"errors"
	"strings"
	"time"

	"rbac/models"
)

func calendarDayUTC(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func assignmentStatus(assignment *models.AMCAssignment) string {
	if assignment == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(assignment.Status))
}

func errIfAMCClosed(assignment *models.AMCAssignment) error {
	if assignmentStatus(assignment) == "closed" {
		return errors.New("cannot update a closed AMC")
	}
	return nil
}

// AMCAssignmentCanClose is true after the end date has passed, or when the
// assignment is already marked expired, and it is not already closed.
func AMCAssignmentCanClose(assignment *models.AMCAssignment, now time.Time) bool {
	if assignment == nil {
		return false
	}
	status := assignmentStatus(assignment)
	if status == "closed" {
		return false
	}
	if status == "expired" {
		return true
	}
	return calendarDayUTC(now).After(calendarDayUTC(assignment.AMCEndDate))
}

// ErrAMCLifecycle is returned when a list request asks for an unknown lifecycle.
var ErrAMCLifecycle = errors.New("lifecycle must be current or expired")

// AMCAssignmentExpired matches the close rule: a closed assignment stays closed,
// and every other assignment is expired once its status says so or its end date
// is before today.
func AMCAssignmentExpired(assignment *models.AMCAssignment, now time.Time) bool {
	return AMCAssignmentCanClose(assignment, now)
}

// FilterAdminAMCLifecycle keeps the full list when lifecycle is omitted.
// current drops expired rows. expired returns only those rows.
func FilterAdminAMCLifecycle(rows []models.AMCAssignment, lifecycle string, now time.Time) ([]models.AMCAssignment, error) {
	switch lifecycle {
	case "":
		return rows, nil
	case "current", "expired":
	default:
		return nil, ErrAMCLifecycle
	}
	return selectAMCLifecycle(rows, lifecycle == "expired", now, false), nil
}

// FilterSupportAMCLifecycle defaults to current. current is the engineer's
// active assignments that are not date-expired. expired is that engineer's
// expired assignments only.
func FilterSupportAMCLifecycle(rows []models.AMCAssignment, lifecycle string, now time.Time) ([]models.AMCAssignment, error) {
	if lifecycle == "" {
		lifecycle = "current"
	}
	switch lifecycle {
	case "current":
		return selectAMCLifecycle(rows, false, now, true), nil
	case "expired":
		return selectAMCLifecycle(rows, true, now, false), nil
	default:
		return nil, ErrAMCLifecycle
	}
}

func selectAMCLifecycle(rows []models.AMCAssignment, wantExpired bool, now time.Time, activeOnly bool) []models.AMCAssignment {
	out := make([]models.AMCAssignment, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		expired := AMCAssignmentExpired(row, now)
		if activeOnly {
			if assignmentStatus(row) == "active" && !expired {
				out = append(out, rows[i])
			}
			continue
		}
		if expired == wantExpired {
			out = append(out, rows[i])
		}
	}
	return out
}
