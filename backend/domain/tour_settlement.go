package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Closing balance uses confirmed money only.
// Closing = credit applied + original disbursement + additional disbursement - verified expenses - adjustments.
// Positive is unused advance credit. Negative is an amount the engineer still owes.
// Adjustments stay zero until a settlement action exists.
func ClosingBalance(creditApplied, originalPaid, additionalPaid, verified, adjustments decimal.Decimal) decimal.Decimal {
	return creditApplied.Add(originalPaid).Add(additionalPaid).Sub(verified).Sub(adjustments)
}

type CreditLot struct {
	OriginTourID uuid.UUID
	Remaining    decimal.Decimal
}

type CreditUse struct {
	OriginTourID uuid.UUID
	Amount       decimal.Decimal
}

// AllocateCredit draws confirmed credit FIFO. Applied never exceeds the gross approval.
func AllocateCredit(lots []CreditLot, gross decimal.Decimal) ([]CreditUse, decimal.Decimal, decimal.Decimal, error) {
	if gross.IsNegative() {
		return nil, decimal.Zero, decimal.Zero, rule("invalid amount")
	}
	left := gross
	uses := make([]CreditUse, 0)
	for _, lot := range lots {
		if !left.GreaterThan(decimal.Zero) {
			break
		}
		if !lot.Remaining.GreaterThan(decimal.Zero) {
			continue
		}
		take := lot.Remaining
		if take.GreaterThan(left) {
			take = left
		}
		uses = append(uses, CreditUse{OriginTourID: lot.OriginTourID, Amount: take})
		left = left.Sub(take)
	}
	applied := gross.Sub(left)
	return uses, applied, left, nil
}

// IsHistoricallyHigh is true only with at least three earlier approved amounts
// and a request more than twice that average.
func IsHistoricallyHigh(requested decimal.Decimal, prior []decimal.Decimal) bool {
	if len(prior) < 3 {
		return false
	}
	sum := decimal.Zero
	for _, amount := range prior {
		sum = sum.Add(amount)
	}
	average := sum.Div(decimal.NewFromInt(int64(len(prior))))
	return requested.GreaterThan(average.Mul(decimal.NewFromInt(2)))
}

type ExpenseLineRules struct {
	Category    string
	BillType    string
	ExpenseDate string
	Description string
	Claimed     decimal.Decimal
	Reference   string
	Reason      string
}

func ValidateExpenseLine(in ExpenseLineRules) error {
	switch in.Category {
	case "FOOD", "LOCAL_TRAVEL", "ACCOMMODATION", "TRAVEL", "OTHER":
	default:
		return rule("expense category is required")
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(in.ExpenseDate)); err != nil {
		return rule("expense date is required")
	}
	description := strings.TrimSpace(in.Description)
	if description == "" {
		return rule("description is required")
	}
	if len(description) > 500 {
		return rule("description is too long")
	}
	if !in.Claimed.GreaterThan(decimal.Zero) {
		return rule("claimed amount must be greater than 0")
	}
	if in.Claimed.Exponent() < -2 || in.Claimed.GreaterThan(maxMoney) {
		return rule("invalid amount")
	}
	reference := strings.TrimSpace(in.Reference)
	reason := strings.TrimSpace(in.Reason)
	switch in.BillType {
	case "BILL":
		if reference == "" {
			return rule("invoice or reference number is required")
		}
		if len(reference) > 100 {
			return rule("reference number is too long")
		}
		if reason != "" {
			return rule("a bill line cannot include a missing-evidence reason")
		}
	case "NON_BILL":
		if reason == "" {
			return rule("a reason is required when there is no bill")
		}
		if len(reason) > 500 {
			return rule("reason is too long")
		}
		if reference != "" {
			return rule("a non-bill line cannot include a reference number")
		}
	default:
		return rule("choose bill or non-bill")
	}
	return nil
}

func ValidateAcceptedAmount(claimed, accepted decimal.Decimal) error {
	if accepted.IsNegative() || accepted.GreaterThan(claimed) {
		return rule("accepted amount cannot exceed the claimed amount")
	}
	if accepted.Exponent() < -2 {
		return rule("invalid amount")
	}
	return nil
}

type SettlementFacts struct {
	HasLines               bool
	AllVerified            bool
	OriginalDisbursed      bool
	Additional             string
	AdditionalDisbursed    bool
	Closing                decimal.Decimal
	LegacyPaidWithoutLines bool
}

// EvaluateSettlement decides whether an Admin may process the tour.
// Paid tours that predate expense lines can still be processed, and they do not mint credit.
func EvaluateSettlement(facts SettlementFacts) (ready bool, reason string, issueCredit bool) {
	if facts.Additional == "PENDING_APPROVAL" {
		return false, "resolve the additional request", false
	}
	if facts.Additional == "APPROVED" && !facts.AdditionalDisbursed {
		return false, "record the additional disbursement", false
	}
	if facts.LegacyPaidWithoutLines && !facts.HasLines && facts.OriginalDisbursed {
		return true, "", false
	}
	if !facts.HasLines {
		return false, "submit expense lines", false
	}
	if !facts.OriginalDisbursed {
		return false, "record the original disbursement", false
	}
	if !facts.AllVerified {
		return false, "verify every expense line", false
	}
	if facts.Closing.IsNegative() {
		return false, "verified expenses exceed the money paid. This tour cannot be processed until that difference is settled", false
	}
	return true, "", facts.Closing.GreaterThan(decimal.Zero)
}

// BlockingStep names the earliest unsettled obligation on an open tour.
func BlockingStep(status string, disbursed bool, additional, lineState string) (step, detail string) {
	switch status {
	case "PENDING_APPROVAL":
		return "approval", "awaiting approval"
	case "APPROVED":
		if !disbursed {
			return "disbursement", "awaiting disbursement"
		}
		if additional == "PENDING_APPROVAL" {
			return "approval", "awaiting additional-money approval"
		}
		return "bill_submission", "awaiting bill submission"
	case "BILL_SUBMITTED":
		if !disbursed {
			return "disbursement", "awaiting disbursement"
		}
		if additional == "PENDING_APPROVAL" {
			return "approval", "awaiting additional-money approval"
		}
		switch lineState {
		case "rejected", "clarification":
			return "discrepancy", "awaiting clarification or correction"
		case "pending":
			return "line_review", "awaiting bill verification"
		default:
			return "final_processing", "awaiting final processing"
		}
	default:
		return "final_processing", "awaiting final processing"
	}
}
