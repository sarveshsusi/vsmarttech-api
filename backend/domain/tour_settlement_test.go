package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestClosingAndCredit(t *testing.T) {
	closing := ClosingBalance(dec("1000"), dec("5000"), dec("0"), dec("4500"), dec("0"))
	if !closing.Equal(dec("1500")) {
		t.Fatalf("closing %s", closing)
	}
	owed := ClosingBalance(dec("0"), dec("1000"), dec("0"), dec("1800"), dec("0"))
	if !owed.Equal(dec("-800")) {
		t.Fatalf("owed %s", owed)
	}
	lots := []CreditLot{{OriginTourID: uuid.New(), Remaining: dec("1000")}}
	uses, applied, net, err := AllocateCredit(lots, dec("600"))
	if err != nil || !applied.Equal(dec("600")) || !net.IsZero() || len(uses) != 1 {
		t.Fatalf("apply %+v %s %s %v", uses, applied, net, err)
	}
	uses, applied, net, err = AllocateCredit(lots, dec("2500"))
	if err != nil || !applied.Equal(dec("1000")) || !net.Equal(dec("1500")) {
		t.Fatalf("partial %s %s", applied, net)
	}
	if IsHistoricallyHigh(dec("100"), []decimal.Decimal{dec("10"), dec("10")}) {
		t.Fatal("new engineer was compared")
	}
	if !IsHistoricallyHigh(dec("100"), []decimal.Decimal{dec("10"), dec("10"), dec("10")}) {
		t.Fatal("expected historical flag")
	}
}

func TestExpenseAndSettlementRules(t *testing.T) {
	err := ValidateExpenseLine(ExpenseLineRules{Category: "FOOD", BillType: "BILL", ExpenseDate: "2026-10-02", Description: "Meals", Claimed: dec("10")})
	if err == nil || err.Error() != "invoice or reference number is required" {
		t.Fatalf("bill: %v", err)
	}
	err = ValidateExpenseLine(ExpenseLineRules{Category: "FOOD", BillType: "NON_BILL", ExpenseDate: "2026-10-02", Description: "Meals", Claimed: dec("10")})
	if err == nil || err.Error() != "a reason is required when there is no bill" {
		t.Fatalf("non-bill: %v", err)
	}
	ready, reason, issue := EvaluateSettlement(SettlementFacts{HasLines: true, AllVerified: false, OriginalDisbursed: true})
	if ready || reason != "verify every expense line" || issue {
		t.Fatalf("unverified %+v %s %v", ready, reason, issue)
	}
	ready, reason, issue = EvaluateSettlement(SettlementFacts{
		HasLines: true, AllVerified: true, OriginalDisbursed: true, Closing: dec("100"),
	})
	if !ready || !issue {
		t.Fatalf("credit %v %s %v", ready, reason, issue)
	}
	ready, _, issue = EvaluateSettlement(SettlementFacts{
		LegacyPaidWithoutLines: true, OriginalDisbursed: true, Closing: dec("100"),
	})
	if !ready || issue {
		t.Fatal("legacy must process without minting credit")
	}
	ready, reason, _ = EvaluateSettlement(SettlementFacts{
		HasLines: true, AllVerified: true, OriginalDisbursed: true, Closing: dec("-1"),
	})
	if ready || reason == "" {
		t.Fatal("negative closing was allowed")
	}
}

func dec(raw string) decimal.Decimal {
	return decimal.RequireFromString(raw)
}
