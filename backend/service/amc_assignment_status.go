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
