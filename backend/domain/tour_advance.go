package domain

import (
	"strings"

	"github.com/shopspring/decimal"

	"rbac/models"
)

var maxMoney = decimal.RequireFromString("9999999999.99")

type RuleError struct {
	Msg string
}

func (e *RuleError) Error() string {
	return e.Msg
}

func rule(msg string) error {
	return &RuleError{Msg: msg}
}

var tourTransitions = map[models.TourAdvanceStatus][]models.TourAdvanceStatus{
	models.TourStatusPendingApproval: {models.TourStatusApproved, models.TourStatusRejected},
	models.TourStatusApproved:        {models.TourStatusBillSubmitted},
	models.TourStatusBillSubmitted:   {models.TourStatusProcessed},
}

func CanTourTransition(from, to models.TourAdvanceStatus) bool {
	for _, next := range tourTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

type TourExpenseInput struct {
	Food          decimal.Decimal
	Local         decimal.Decimal
	Accommodation decimal.Decimal
	Travel        decimal.Decimal
}

func RequestedAmount(in TourExpenseInput) (decimal.Decimal, error) {
	parts := []decimal.Decimal{in.Food, in.Local, in.Accommodation, in.Travel}
	total := decimal.Zero
	for _, part := range parts {
		if part.IsNegative() {
			return decimal.Zero, rule("expenses cannot be negative")
		}
		if part.GreaterThan(maxMoney) {
			return decimal.Zero, rule("amount is too large")
		}
		if part.Exponent() < -2 {
			return decimal.Zero, rule("invalid amount")
		}
		total = total.Add(part)
	}
	if total.GreaterThan(maxMoney) {
		return decimal.Zero, rule("amount is too large")
	}
	return total, nil
}

func ParseMoney(raw string) (decimal.Decimal, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return decimal.Zero, rule("amount is required")
	}
	amount, err := decimal.NewFromString(trimmed)
	if err != nil {
		return decimal.Zero, rule("invalid amount")
	}
	if amount.IsNegative() {
		return decimal.Zero, rule("expenses cannot be negative")
	}
	if amount.Exponent() < -2 {
		return decimal.Zero, rule("invalid amount")
	}
	if amount.GreaterThan(maxMoney) {
		return decimal.Zero, rule("amount is too large")
	}
	return amount, nil
}

type TourCreateRules struct {
	WorkType   models.TourWorkType
	PONumber   string
	HasTicket  bool
	Location   string
	Persons    int
	Days       int
	TravelMode models.TourTravelMode
	Expenses   TourExpenseInput
}

func ValidateTourCreate(in TourCreateRules) error {
	if strings.TrimSpace(in.Location) == "" {
		return rule("location is required")
	}
	if in.Persons <= 0 {
		return rule("persons travelling must be greater than 0")
	}
	if in.Days <= 0 {
		return rule("days planned must be greater than 0")
	}
	switch in.TravelMode {
	case models.TourTravelBus, models.TourTravelTrain, models.TourTravelCar:
	default:
		return rule("travel mode is required")
	}

	po := strings.TrimSpace(in.PONumber)
	switch in.WorkType {
	case models.TourWorkInstallation:
		if po == "" {
			return rule("PO number is required for installation")
		}
		if len(po) > 100 {
			return rule("PO number is too long")
		}
		if in.HasTicket {
			return rule("installation cannot include a ticket")
		}
	case models.TourWorkService:
		if !in.HasTicket {
			return rule("ticket is required for service")
		}
		if po != "" {
			return rule("service cannot include a PO number")
		}
	default:
		return rule("work type is required")
	}

	if _, err := RequestedAmount(in.Expenses); err != nil {
		return err
	}
	return nil
}

func ValidateApprovedAmount(requested, approved decimal.Decimal) error {
	if approved.LessThanOrEqual(decimal.Zero) {
		return rule("approved amount must be greater than 0")
	}
	if approved.Exponent() < -2 {
		return rule("invalid amount")
	}
	if approved.GreaterThan(requested) {
		return rule("approved amount cannot exceed the requested amount")
	}
	return nil
}
