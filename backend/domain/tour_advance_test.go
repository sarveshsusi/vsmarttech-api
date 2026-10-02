package domain

import (
	"testing"

	"github.com/shopspring/decimal"

	"rbac/models"
)

func TestCanTourTransition(t *testing.T) {
	allowed := [][2]models.TourAdvanceStatus{
		{models.TourStatusPendingApproval, models.TourStatusApproved},
		{models.TourStatusPendingApproval, models.TourStatusRejected},
		{models.TourStatusApproved, models.TourStatusBillSubmitted},
		{models.TourStatusBillSubmitted, models.TourStatusProcessed},
	}
	for _, pair := range allowed {
		if !CanTourTransition(pair[0], pair[1]) {
			t.Fatalf("expected %s -> %s", pair[0], pair[1])
		}
	}

	blocked := [][2]models.TourAdvanceStatus{
		{models.TourStatusPendingApproval, models.TourStatusBillSubmitted},
		{models.TourStatusPendingApproval, models.TourStatusProcessed},
		{models.TourStatusApproved, models.TourStatusProcessed},
		{models.TourStatusRejected, models.TourStatusApproved},
		{models.TourStatusRejected, models.TourStatusBillSubmitted},
		{models.TourStatusProcessed, models.TourStatusApproved},
		{models.TourStatusBillSubmitted, models.TourStatusApproved},
		{models.TourStatusBillSubmitted, models.TourStatusBillSubmitted},
		{models.TourStatusProcessed, models.TourStatusProcessed},
	}
	for _, pair := range blocked {
		if CanTourTransition(pair[0], pair[1]) {
			t.Fatalf("did not expect %s -> %s", pair[0], pair[1])
		}
	}
}

func TestRequestedAmountIgnoresNothingAndSums(t *testing.T) {
	total, err := RequestedAmount(TourExpenseInput{
		Food:          decimal.RequireFromString("2700"),
		Local:         decimal.RequireFromString("1000"),
		Accommodation: decimal.RequireFromString("4500"),
		Travel:        decimal.RequireFromString("2000"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !total.Equal(decimal.RequireFromString("10200")) {
		t.Fatalf("total = %s", total)
	}
}

func TestRequestedAmountRejectsNegative(t *testing.T) {
	_, err := RequestedAmount(TourExpenseInput{
		Food:          decimal.RequireFromString("-1"),
		Local:         decimal.Zero,
		Accommodation: decimal.Zero,
		Travel:        decimal.Zero,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateApprovedAmount(t *testing.T) {
	requested := decimal.RequireFromString("10200")
	if err := ValidateApprovedAmount(requested, decimal.RequireFromString("10000")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateApprovedAmount(requested, decimal.RequireFromString("10200.01")); err == nil {
		t.Fatal("expected amount above requested to fail")
	}
	if err := ValidateApprovedAmount(requested, decimal.Zero); err == nil {
		t.Fatal("expected zero to fail")
	}
}

func TestValidateTourCreateRules(t *testing.T) {
	base := validCreateRules()
	if err := ValidateTourCreate(base); err != nil {
		t.Fatal(err)
	}

	noPO := base
	noPO.PONumber = "  "
	if err := ValidateTourCreate(noPO); err == nil {
		t.Fatal("expected PO required")
	}

	withTicket := base
	withTicket.HasTicket = true
	if err := ValidateTourCreate(withTicket); err == nil || err.Error() != "installation cannot include a ticket" {
		t.Fatalf("expected installation ticket rejection, got %v", err)
	}

	service := base
	service.WorkType = models.TourWorkService
	service.PONumber = ""
	service.HasTicket = true
	if err := ValidateTourCreate(service); err != nil {
		t.Fatal(err)
	}

	servicePO := service
	servicePO.PONumber = "PO-1"
	if err := ValidateTourCreate(servicePO); err == nil {
		t.Fatal("expected service PO rejection")
	}

	noTicket := service
	noTicket.HasTicket = false
	noTicket.PONumber = ""
	if err := ValidateTourCreate(noTicket); err == nil {
		t.Fatal("expected ticket required")
	}

	base.Persons = 0
	if err := ValidateTourCreate(base); err == nil {
		t.Fatal("expected persons > 0")
	}
	base.Persons = 3
	base.Days = 0
	if err := ValidateTourCreate(base); err == nil {
		t.Fatal("expected days > 0")
	}
}

func validCreateRules() TourCreateRules {
	return TourCreateRules{
		WorkType:   models.TourWorkInstallation,
		PONumber:   "THPG26/263410240",
		Location:   "TVARUR",
		Persons:    3,
		Days:       3,
		TravelMode: models.TourTravelCar,
		Expenses: TourExpenseInput{
			Food:          decimal.RequireFromString("2700"),
			Local:         decimal.RequireFromString("1000"),
			Accommodation: decimal.RequireFromString("4500"),
			Travel:        decimal.RequireFromString("2000"),
		},
	}
}
