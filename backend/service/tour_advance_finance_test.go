package service

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"rbac/domain"
	"rbac/models"
)

func TestTourAdvanceEligibilityAndLedger(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)

	first := mustCreateTour(t, svc, fx)
	_, err := svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
	var blocked *TourBlockedError
	if !errors.As(err, &blocked) || blocked.Step != "approval" || blocked.RequestNumber != first.RequestNumber {
		t.Fatalf("pending block: %v", err)
	}
	if _, err := svc.Reject(fx.superAdmin.ID, models.RoleSuperAdmin, first.ID, "no payment"); err != nil {
		t.Fatal(err)
	}
	second := mustCreateTour(t, svc, fx)
	if _, err := svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID)); err == nil {
		t.Fatal("rejected-without-payment should be the only case that stays eligible, and this tour is still pending")
	}

	approved := mustApprove(t, svc, fx, second.ID, "1000")
	if approved.PaymentMode != "" || approved.NetPayable == nil || *approved.NetPayable != "1000.00" {
		t.Fatalf("approved is not unpaid: %+v", approved)
	}
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
	if !errors.As(err, &blocked) || blocked.Step != "disbursement" {
		t.Fatalf("unpaid block: %v", err)
	}
	if _, err := svc.Disburse(fx.admin.ID, models.RoleAdmin, second.ID, "ORIGINAL", "CASH", ""); !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("admin disburse: %v", err)
	}
	if _, err := svc.Disburse(fx.superAdmin.ID, models.RoleSuperAdmin, second.ID, "ORIGINAL", "ONLINE", ""); err == nil {
		t.Fatal("online without transaction id")
	}
	paid, err := svc.Disburse(fx.superAdmin.ID, models.RoleSuperAdmin, second.ID, "ORIGINAL", "CASH", "")
	if err != nil {
		t.Fatal(err)
	}
	again, err := svc.Disburse(fx.superAdmin.ID, models.RoleSuperAdmin, second.ID, "ORIGINAL", "CASH", "")
	if err != nil || again.PaymentMode != "CASH" {
		t.Fatalf("retry disburse: %v", err)
	}
	var disburseCount int64
	if err := db.Model(&models.TourAdvanceDisbursement{}).Where("tour_advance_request_id = ?", second.ID).Count(&disburseCount).Error; err != nil {
		t.Fatal(err)
	}
	if disburseCount != 1 {
		t.Fatalf("disbursements %d", disburseCount)
	}
	_ = paid

	billed, err := svc.SubmitExpenses(fx.engineer.ID, models.RoleSupport, second.ID, []ExpenseLineInput{{
		Category: "LOCAL_TRAVEL", BillType: "NON_BILL", ExpenseDate: "2026-10-03",
		Description: "Auto", ClaimedAmount: "200", MissingReason: "driver had no receipt",
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
	if !errors.As(err, &blocked) || blocked.Step != "line_review" {
		t.Fatalf("review block: %v", err)
	}
	expenseID := uuid.MustParse(billed.Expenses[0].ID)
	if _, err := svc.ReviewExpense(fx.superAdmin.ID, models.RoleSuperAdmin, second.ID, expenseID, "ACCEPT", "200", ""); !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("super review: %v", err)
	}
	if _, err := svc.ReviewExpense(fx.admin.ID, models.RoleAdmin, second.ID, expenseID, "REJECT", "", "need a clearer reason"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Process(fx.admin.ID, models.RoleAdmin, second.ID)
	assertRule(t, err, "verify every expense line")
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
	if !errors.As(err, &blocked) || blocked.Step != "discrepancy" {
		t.Fatalf("dispute block: %v", err)
	}
	if _, err := svc.AmendExpense(fx.otherEngineer.ID, models.RoleSupport, second.ID, expenseID, sampleBill()); !errors.Is(err, ErrTourNotFound) {
		t.Fatalf("other amend: %v", err)
	}
	amended, err := svc.AmendExpense(fx.engineer.ID, models.RoleSupport, second.ID, expenseID, ExpenseLineInput{
		Category: "FOOD", BillType: "BILL", ExpenseDate: "2026-10-03", Description: "Dinner", ClaimedAmount: "200", ReferenceNumber: "INV-9",
	})
	if err != nil || amended.Expenses[0].Status != string(models.TourExpensePendingReview) {
		t.Fatalf("amend: %v %+v", err, amended.Expenses)
	}
	if _, err := svc.ReviewExpense(fx.admin.ID, models.RoleAdmin, second.ID, expenseID, "ACCEPT", "150", "rounded"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AmendExpense(fx.engineer.ID, models.RoleSupport, second.ID, expenseID, sampleBill()); err == nil {
		t.Fatal("verified line was editable")
	}
	if _, err := svc.Process(fx.superAdmin.ID, models.RoleSuperAdmin, second.ID); !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("super process: %v", err)
	}
	processed, err := svc.Process(fx.admin.ID, models.RoleAdmin, second.ID)
	if err != nil || processed.Status != string(models.TourStatusProcessed) {
		t.Fatalf("process: %v", err)
	}
	var issued int64
	if err := db.Model(&models.TourAdvanceLedgerEntry{}).Where("originating_tour_id = ? AND entry_type = ?", second.ID, "CREDIT_ISSUED").Count(&issued).Error; err != nil {
		t.Fatal(err)
	}
	if issued != 1 {
		t.Fatalf("credits issued %d", issued)
	}
	if _, err := svc.Process(fx.admin.ID, models.RoleAdmin, second.ID); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("second process: %v", err)
	}
	if err := db.Model(&models.TourAdvanceLedgerEntry{}).Where("entry_type = ?", "CREDIT_ISSUED").Count(&issued).Error; err != nil {
		t.Fatal(err)
	}
	if issued != 1 {
		t.Fatalf("retry minted credit %d", issued)
	}

	third := mustCreateTour(t, svc, fx)
	covered := mustApprove(t, svc, fx, third.ID, "600")
	if covered.CreditApplied != "600.00" || covered.NetPayable == nil || *covered.NetPayable != "0.00" || covered.PaymentMode != "CARRY_FORWARD" {
		t.Fatalf("carry forward %+v net %+v mode %s", covered.CreditApplied, covered.NetPayable, covered.PaymentMode)
	}
	var applied int64
	if err := db.Model(&models.TourAdvanceLedgerEntry{}).Where("consuming_tour_id = ? AND entry_type = ?", third.ID, "CREDIT_APPLIED").Count(&applied).Error; err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("applied rows %d", applied)
	}
	_, pool, err := creditLots(db, engineerID(t, db, fx.engineer.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !pool.Equal(decimal.RequireFromString("250")) {
		t.Fatalf("remaining credit %s", pool)
	}
	if _, err := svc.Disburse(fx.superAdmin.ID, models.RoleSuperAdmin, third.ID, "ORIGINAL", "CASH", ""); err == nil {
		t.Fatal("zero net still asked for a payment")
	}

	summary, err := svc.FinancialSummary(fx.admin.ID, models.RoleAdmin, TourSummaryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalDisbursed != "1000.00" {
		t.Fatalf("disbursed total %s", summary.TotalDisbursed)
	}
	if _, err := svc.FinancialSummary(fx.engineer.ID, models.RoleSupport, TourSummaryFilter{}); !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("support summary: %v", err)
	}
	history, err := svc.OwnHistory(fx.engineer.ID, models.RoleSupport)
	if err != nil || history.AvailableCredit != "250.00" {
		t.Fatalf("history: %v %+v", err, history)
	}
	if _, err := svc.EngineerHistory(fx.otherEngineer.ID, models.RoleSupport, engineerID(t, db, fx.engineer.ID)); !errors.Is(err, ErrTourNotFound) {
		t.Fatalf("other history: %v", err)
	}
}

func TestTourAdvanceOverspendBlocksAndLegacyProcesses(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)

	created := mustCreateTour(t, svc, fx)
	mustApprove(t, svc, fx, created.ID, "1000")
	if _, err := svc.Disburse(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "ORIGINAL", "ONLINE", "UTR-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateAdditional(fx.engineer.ID, models.RoleSupport, created.ID, "300", "more stay"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApproveAdditional(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "300", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Disburse(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "ADDITIONAL", "CASH", ""); err != nil {
		t.Fatal(err)
	}
	billed, err := svc.SubmitExpenses(fx.engineer.ID, models.RoleSupport, created.ID, []ExpenseLineInput{{
		Category: "ACCOMMODATION", BillType: "BILL", ExpenseDate: "2026-10-04",
		Description: "Hotel", ClaimedAmount: "2000", ReferenceNumber: "H-1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	expenseID := uuid.MustParse(billed.Expenses[0].ID)
	if _, err := svc.ReviewExpense(fx.admin.ID, models.RoleAdmin, created.ID, expenseID, "ACCEPT", "2000", ""); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Process(fx.admin.ID, models.RoleAdmin, created.ID)
	assertRule(t, err, "verified expenses exceed the money paid. This tour cannot be processed until that difference is settled")
	var credits int64
	if err := db.Model(&models.TourAdvanceLedgerEntry{}).Count(&credits).Error; err != nil || credits != 0 {
		t.Fatalf("partial ledger %d %v", credits, err)
	}

	legacy := mustCreateTour(t, svc, fx.other())
	if err := db.Model(&models.TourAdvanceRequest{}).Where("id = ?", legacy.ID).Updates(map[string]interface{}{
		"status": models.TourStatusBillSubmitted, "approved_amount": decimal.RequireFromString("500"),
		"net_payable": decimal.RequireFromString("500"), "payment_mode": "CASH",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Process(fx.admin.ID, models.RoleAdmin, legacy.ID); err != nil {
		t.Fatalf("legacy process: %v", err)
	}
	if err := db.Model(&models.TourAdvanceLedgerEntry{}).Where("originating_tour_id = ?", legacy.ID).Count(&credits).Error; err != nil || credits != 0 {
		t.Fatalf("legacy credit %d %v", credits, err)
	}
}

func TestTourAdvanceFlagRequiresJustification(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)
	if _, err := svc.UpdatePolicy(fx.superAdmin.ID, models.RoleSuperAdmin, strPtr("100")); !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("super policy: %v", err)
	}
	if _, err := svc.UpdatePolicy(fx.admin.ID, models.RoleAdmin, strPtr("100")); err != nil {
		t.Fatal(err)
	}
	created := mustCreateTour(t, svc, fx)
	if len(created.Flags) == 0 {
		t.Fatal("expected a high value flag")
	}
	_, err := svc.Approve(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "1000", "", "")
	assertRule(t, err, "a justification is required for this flagged request")
	var ledger int64
	if err := db.Model(&models.TourAdvanceLedgerEntry{}).Count(&ledger).Error; err != nil || ledger != 0 {
		t.Fatalf("failed approval wrote ledger %d %v", ledger, err)
	}
	if _, err := svc.Approve(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "1000", "", "site is remote"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentTourCreate(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	success := 0
	blocked := 0
	for err := range errs {
		if err == nil {
			success++
			continue
		}
		var block *TourBlockedError
		if errors.As(err, &block) || isUniqueViolation(err) {
			blocked++
			continue
		}
		t.Fatal(err)
	}
	if success != 1 || blocked != 1 {
		t.Fatalf("success %d blocked %d", success, blocked)
	}
}

func mustCreateTour(t *testing.T, svc *TourAdvanceService, fx tourFixture) *TourAdvanceView {
	t.Helper()
	created, err := svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func mustApprove(t *testing.T, svc *TourAdvanceService, fx tourFixture, id uuid.UUID, amount string) *TourAdvanceView {
	t.Helper()
	view, err := svc.Approve(fx.superAdmin.ID, models.RoleSuperAdmin, id, amount, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func engineerID(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	var engineer models.SupportEngineer
	if err := db.Where("user_id = ?", userID).First(&engineer).Error; err != nil {
		t.Fatal(err)
	}
	return engineer.ID
}

func (fx tourFixture) other() tourFixture {
	fx.engineer = fx.otherEngineer
	return fx
}

func strPtr(value string) *string { return &value }

func TestUnverifiedExpenseDoesNotCreateCredit(t *testing.T) {
	closing := domain.ClosingBalance(decimal.Zero, decimal.RequireFromString("1000"), decimal.Zero, decimal.Zero, decimal.Zero)
	if !closing.Equal(decimal.RequireFromString("1000")) {
		t.Fatal(closing)
	}
}
