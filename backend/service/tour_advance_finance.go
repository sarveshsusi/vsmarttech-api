package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"rbac/domain"
	"rbac/models"
)

type TourBlockedError struct {
	RequestID     uuid.UUID
	RequestNumber string
	Step          string
	Msg           string
}

func (e *TourBlockedError) Error() string { return e.Msg }

type ExpenseLineInput struct {
	Category        string
	BillType        string
	ExpenseDate     string
	Description     string
	ClaimedAmount   string
	ReferenceNumber string
	MissingReason   string
}

type EligibilityView struct {
	Eligible        bool   `json:"eligible"`
	Message         string `json:"message,omitempty"`
	RequestID       string `json:"request_id,omitempty"`
	RequestNumber   string `json:"request_number,omitempty"`
	Step            string `json:"step,omitempty"`
	AvailableCredit string `json:"available_credit"`
}

type TourFlagView struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type TourExpenseView struct {
	ID              string     `json:"id"`
	Category        string     `json:"category"`
	BillType        string     `json:"bill_type"`
	ExpenseDate     string     `json:"expense_date"`
	Description     string     `json:"description"`
	ClaimedAmount   string     `json:"claimed_amount"`
	ReferenceNumber string     `json:"reference_number,omitempty"`
	MissingReason   string     `json:"missing_reason,omitempty"`
	AcceptedAmount  *string    `json:"accepted_amount,omitempty"`
	Status          string     `json:"status"`
	ReviewReason    string     `json:"review_reason,omitempty"`
	ReviewedByName  string     `json:"reviewed_by_name,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
}

type TourDisbursementView struct {
	Source        string `json:"source"`
	Amount        string `json:"amount"`
	PaymentMode   string `json:"payment_mode"`
	TransactionID string `json:"transaction_id,omitempty"`
}

type TourLedgerView struct {
	EntryType         string    `json:"entry_type"`
	Amount            string    `json:"amount"`
	BalanceBefore     string    `json:"balance_before"`
	BalanceAfter      string    `json:"balance_after"`
	OriginatingTourID string    `json:"originating_tour_id"`
	ConsumingTourID   string    `json:"consuming_tour_id,omitempty"`
	Reason            string    `json:"reason"`
	CreatedAt         time.Time `json:"created_at"`
}

type EngineerBalance struct {
	EngineerID   string `json:"engineer_id"`
	EngineerName string `json:"engineer_name"`
	Balance      string `json:"balance"`
}

type TourFinancialSummary struct {
	TotalRequested       string            `json:"total_requested"`
	OriginalApproved     string            `json:"original_approved"`
	AdditionalRequested  string            `json:"additional_requested"`
	AdditionalApproved   string            `json:"additional_approved"`
	TotalDisbursed       string            `json:"total_disbursed"`
	BillClaimed          string            `json:"bill_claimed"`
	BillVerified         string            `json:"bill_verified"`
	NonBillClaimed       string            `json:"non_bill_claimed"`
	NonBillVerified      string            `json:"non_bill_verified"`
	TotalVerified        string            `json:"total_verified"`
	Outstanding          string            `json:"outstanding"`
	UnsettledTours       int64             `json:"unsettled_tours"`
	AwaitingVerification int64             `json:"awaiting_verification"`
	RejectedOrIncomplete int64             `json:"rejected_or_incomplete"`
	HighValueRequests    int64             `json:"high_value_requests"`
	BlockedEngineers     int64             `json:"blocked_engineers"`
	CarryForward         []EngineerBalance `json:"carry_forward"`
}

type TourSummaryFilter struct {
	EngineerID    string
	CustomerID    string
	CompanyID     string
	Location      string
	RequestNumber string
	From          string
	To            string
	Status        string
	Verification  string
}

type EngineerFinanceHistory struct {
	EngineerID      string            `json:"engineer_id"`
	EngineerName    string            `json:"engineer_name"`
	AvailableCredit string            `json:"available_credit"`
	Tours           []TourAdvanceView `json:"tours"`
	Ledger          []TourLedgerView  `json:"ledger"`
}

type TourPolicyView struct {
	HighValueThreshold *string `json:"high_value_threshold"`
}

func lockEngineer(tx *gorm.DB, engineerID uuid.UUID) error {
	if tx.Dialector.Name() == "postgres" {
		var engineer models.SupportEngineer
		return tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&engineer, "id = ?", engineerID).Error
	}
	return tx.Model(&models.SupportEngineer{}).Where("id = ?", engineerID).Update("updated_at", time.Now()).Error
}

func (s *TourAdvanceService) ensureEligible(tx *gorm.DB, engineerID uuid.UUID) error {
	var open models.TourAdvanceRequest
	err := tx.Preload("Additional").Preload("Expenses").Preload("Disbursements").
		Where("engineer_id = ? AND status NOT IN ?", engineerID, []models.TourAdvanceStatus{
			models.TourStatusRejected, models.TourStatusProcessed,
		}).Order("created_at ASC").First(&open).Error
	if err == nil {
		return blockedFrom(&open)
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	var paidRejected models.TourAdvanceRequest
	err = tx.Where("engineer_id = ? AND status = ? AND (payment_mode IS NOT NULL OR id IN (SELECT tour_advance_request_id FROM tour_advance_disbursements))",
		engineerID, models.TourStatusRejected).First(&paidRejected).Error
	if err == nil {
		return &TourBlockedError{
			RequestID: paidRejected.ID, RequestNumber: paidRejected.RequestNumber, Step: "final_processing",
			Msg: fmt.Sprintf("%s was rejected after a payment was recorded. Settle that payment before requesting another advance.", paidRejected.RequestNumber),
		}
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return nil
}

func blockedFrom(row *models.TourAdvanceRequest) error {
	disbursed := row.PaymentMode != nil || hasDisbursement(row.Disbursements, "ORIGINAL") || (row.NetPayable != nil && row.NetPayable.IsZero() && row.CreditApplied.GreaterThan(decimal.Zero))
	additional := ""
	if row.Additional != nil {
		additional = string(row.Additional.Status)
	}
	step, detail := domain.BlockingStep(string(row.Status), disbursed, additional, lineState(row.Expenses))
	return &TourBlockedError{
		RequestID: row.ID, RequestNumber: row.RequestNumber, Step: step,
		Msg: fmt.Sprintf("%s is %s. Finish that settlement before requesting another advance.", row.RequestNumber, detail),
	}
}

func lineState(lines []models.TourAdvanceExpense) string {
	state := "verified"
	for _, line := range lines {
		switch line.Status {
		case models.TourExpenseRejected:
			return "rejected"
		case models.TourExpenseNeedsClarification:
			return "clarification"
		case models.TourExpensePendingReview:
			state = "pending"
		}
	}
	if len(lines) == 0 {
		return "none"
	}
	return state
}

func hasDisbursement(rows []models.TourAdvanceDisbursement, source string) bool {
	for _, row := range rows {
		if row.Source == source {
			return true
		}
	}
	return false
}

func (s *TourAdvanceService) noteBlocked(actor uuid.UUID, err error) {
	blocked, ok := err.(*TourBlockedError)
	if !ok || s.db == nil {
		return
	}
	_ = writeTourEvent(s.db, blocked.RequestID, models.TourEventBlocked, actor, blocked.Msg)
	s.notifyEngineer(blocked.RequestID, models.NotificationTypeTourAdvanceBlocked, "New advance blocked", blocked.Msg)
}

func creditLots(tx *gorm.DB, engineerID uuid.UUID) ([]domain.CreditLot, decimal.Decimal, error) {
	var issued []models.TourAdvanceLedgerEntry
	if err := tx.Where("engineer_id = ? AND entry_type = ?", engineerID, "CREDIT_ISSUED").
		Order("created_at ASC").Find(&issued).Error; err != nil {
		return nil, decimal.Zero, err
	}
	var applied []models.TourAdvanceLedgerEntry
	if err := tx.Where("engineer_id = ? AND entry_type = ?", engineerID, "CREDIT_APPLIED").
		Find(&applied).Error; err != nil {
		return nil, decimal.Zero, err
	}
	used := map[uuid.UUID]decimal.Decimal{}
	pool := decimal.Zero
	for _, row := range applied {
		used[row.OriginatingTourID] = used[row.OriginatingTourID].Add(row.Amount)
		pool = pool.Sub(row.Amount)
	}
	lots := make([]domain.CreditLot, 0, len(issued))
	for _, row := range issued {
		pool = pool.Add(row.Amount)
		remaining := row.Amount.Sub(used[row.OriginatingTourID])
		if remaining.IsNegative() {
			remaining = decimal.Zero
		}
		lots = append(lots, domain.CreditLot{OriginTourID: row.OriginatingTourID, Remaining: remaining})
	}
	if pool.IsNegative() {
		pool = decimal.Zero
	}
	return lots, pool, nil
}

func writeCreditUses(tx *gorm.DB, engineerID, consuming uuid.UUID, actor uuid.UUID, uses []domain.CreditUse, pool decimal.Decimal) error {
	balance := pool
	for _, use := range uses {
		before := balance
		after := balance.Sub(use.Amount)
		key := fmt.Sprintf("credit-applied:%s:%s", consuming, use.OriginTourID)
		var existing int64
		if err := tx.Model(&models.TourAdvanceLedgerEntry{}).Where("idempotency_key = ?", key).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			continue
		}
		consumingID := consuming
		if err := tx.Create(&models.TourAdvanceLedgerEntry{
			ID: uuid.New(), EngineerID: engineerID, OriginatingTourID: use.OriginTourID, ConsumingTourID: &consumingID,
			EntryType: "CREDIT_APPLIED", Amount: use.Amount, BalanceBefore: before, BalanceAfter: after,
			ActorUserID: actor, Reason: "Applied to the next approved advance", IdempotencyKey: key, CreatedAt: time.Now(),
		}).Error; err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return err
		}
		balance = after
	}
	return nil
}

func (s *TourAdvanceService) Disburse(actor uuid.UUID, role models.Role, id uuid.UUID, source, paymentMode, transactionID string) (*TourAdvanceView, error) {
	if role != models.RoleSuperAdmin {
		return nil, ErrTourForbidden
	}
	source = strings.ToUpper(strings.TrimSpace(source))
	if source != "ORIGINAL" && source != "ADDITIONAL" {
		return nil, &domain.RuleError{Msg: "invalid disbursement"}
	}
	mode, txn, err := domain.NormalizePayment(paymentMode, transactionID)
	if err != nil {
		return nil, err
	}
	var becameReady bool
	err = s.db.Transaction(func(tx *gorm.DB) error {
		row, err := lockTour(tx, id)
		if err != nil {
			return err
		}
		if row.Status == models.TourStatusRejected || row.Status == models.TourStatusPendingApproval || row.Status == models.TourStatusProcessed {
			return ErrInvalidTransition
		}
		expected := decimal.Zero
		if source == "ORIGINAL" {
			if row.NetPayable == nil {
				return ErrInvalidTransition
			}
			if row.NetPayable.IsZero() {
				return &domain.RuleError{Msg: "no payment is due"}
			}
			expected = *row.NetPayable
		} else {
			extra, err := lockAdditional(tx, id)
			if err != nil {
				return err
			}
			if extra.Status != models.TourAdditionalApproved || extra.ApprovedAmount == nil {
				return ErrInvalidTransition
			}
			expected = *extra.ApprovedAmount
		}
		var existing models.TourAdvanceDisbursement
		findErr := tx.Where("tour_advance_request_id = ? AND source = ?", id, source).First(&existing).Error
		if findErr == nil {
			currentTxn := ""
			if existing.TransactionID != nil {
				currentTxn = *existing.TransactionID
			}
			if existing.PaymentMode == mode && currentTxn == txn && existing.Amount.Equal(expected) {
				return nil
			}
			return &domain.RuleError{Msg: "payment is already recorded"}
		}
		if findErr != gorm.ErrRecordNotFound {
			return findErr
		}
		var txnPtr *string
		if txn != "" {
			txnPtr = &txn
		}
		if err := tx.Create(&models.TourAdvanceDisbursement{
			ID: uuid.New(), TourAdvanceRequestID: id, Source: source, Amount: expected,
			PaymentMode: mode, TransactionID: txnPtr, ActorUserID: actor,
			IdempotencyKey: fmt.Sprintf("disburse:%s:%s", id, source), CreatedAt: time.Now(),
		}).Error; err != nil {
			if isUniqueViolation(err) {
				return nil
			}
			return err
		}
		now := time.Now()
		updates := map[string]interface{}{"payment_mode": mode, "updated_at": now}
		if txn == "" {
			updates["transaction_id"] = nil
		} else {
			updates["transaction_id"] = txn
		}
		if source == "ORIGINAL" {
			if err := tx.Model(&models.TourAdvanceRequest{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&models.TourAdvanceAdditional{}).Where("tour_advance_request_id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		if err := writeTourEvent(tx, id, models.TourEventDisbursed, actor, source+" "+expected.StringFixed(2)+" "+mode); err != nil {
			return err
		}
		ready, err := settlementReady(tx, id)
		if err != nil {
			return err
		}
		becameReady = ready
		if ready {
			return markReady(tx, id, actor)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyEngineer(id, models.NotificationTypeTourAdvanceDisbursed, "Tour advance paid",
		fmt.Sprintf("Payment for %s was recorded.", view.RequestNumber))
	if becameReady {
		s.notifyRole(models.RoleAdmin, models.NotificationTypeTourAdvanceReady, "Tour ready to process",
			fmt.Sprintf("%s is ready for final processing.", view.RequestNumber), &models.TourAdvanceRequest{ID: id, RequestNumber: view.RequestNumber})
	}
	return view, nil
}

func lockTour(tx *gorm.DB, id uuid.UUID) (*models.TourAdvanceRequest, error) {
	q := tx
	if tx.Dialector.Name() == "postgres" {
		q = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row models.TourAdvanceRequest
	err := q.First(&row, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrTourNotFound
	}
	return &row, err
}

func settlementReady(tx *gorm.DB, id uuid.UUID) (bool, error) {
	var row models.TourAdvanceRequest
	if err := tx.Preload("Additional").Preload("Expenses").Preload("Disbursements").First(&row, "id = ?", id).Error; err != nil {
		return false, err
	}
	if row.Status != models.TourStatusBillSubmitted {
		return false, nil
	}
	ready, _, _ := domain.EvaluateSettlement(factsFrom(&row))
	return ready, nil
}

func factsFrom(row *models.TourAdvanceRequest) domain.SettlementFacts {
	originalPaid := decimal.Zero
	additionalPaid := decimal.Zero
	originalDisbursed := row.PaymentMode != nil
	additionalDisbursed := row.Additional != nil && row.Additional.PaymentMode != nil
	for _, item := range row.Disbursements {
		if item.Source == "ORIGINAL" {
			originalDisbursed = true
			originalPaid = item.Amount
		}
		if item.Source == "ADDITIONAL" {
			additionalDisbursed = true
			additionalPaid = item.Amount
		}
	}
	if originalPaid.IsZero() && row.PaymentMode != nil && row.NetPayable != nil {
		originalPaid = *row.NetPayable
	}
	verified := decimal.Zero
	allVerified := len(row.Expenses) > 0
	for _, line := range row.Expenses {
		if line.Status != models.TourExpenseVerified || line.AcceptedAmount == nil {
			allVerified = false
			continue
		}
		verified = verified.Add(*line.AcceptedAmount)
	}
	additional := ""
	if row.Additional != nil {
		additional = string(row.Additional.Status)
		if additional == string(models.TourAdditionalApproved) && row.Additional.ApprovedAmount != nil && additionalPaid.IsZero() && additionalDisbursed {
			additionalPaid = *row.Additional.ApprovedAmount
		}
	}
	closing := domain.ClosingBalance(row.CreditApplied, originalPaid, additionalPaid, verified, decimal.Zero)
	legacy := len(row.Expenses) == 0 && originalDisbursed
	return domain.SettlementFacts{
		HasLines: len(row.Expenses) > 0, AllVerified: allVerified, OriginalDisbursed: originalDisbursed,
		Additional: additional, AdditionalDisbursed: additionalDisbursed || additional != string(models.TourAdditionalApproved),
		Closing: closing, LegacyPaidWithoutLines: legacy,
	}
}

func markReady(tx *gorm.DB, id, actor uuid.UUID) error {
	var count int64
	if err := tx.Model(&models.TourAdvanceEvent{}).Where("tour_advance_request_id = ? AND action = ?", id, models.TourEventReady).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return writeTourEvent(tx, id, models.TourEventReady, actor, "")
}

func (s *TourAdvanceService) Eligibility(actor uuid.UUID, role models.Role) (*EligibilityView, error) {
	if role != models.RoleSupport {
		return nil, ErrTourForbidden
	}
	var engineer models.SupportEngineer
	if err := s.db.Where("user_id = ? AND is_active = ?", actor, true).First(&engineer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTourForbidden
		}
		return nil, err
	}
	_, pool, err := creditLots(s.db, engineer.ID)
	if err != nil {
		return nil, err
	}
	view := &EligibilityView{Eligible: true, AvailableCredit: pool.StringFixed(2)}
	if err := s.ensureEligible(s.db, engineer.ID); err != nil {
		blocked, ok := err.(*TourBlockedError)
		if !ok {
			return nil, err
		}
		view.Eligible = false
		view.Message = blocked.Msg
		view.RequestID = blocked.RequestID.String()
		view.RequestNumber = blocked.RequestNumber
		view.Step = blocked.Step
	}
	return view, nil
}

func (s *TourAdvanceService) recordFlags(tx *gorm.DB, tourID, engineerID, actor uuid.UUID, amount decimal.Decimal, additional bool) error {
	flags, err := s.collectFlags(tx, engineerID, amount, additional)
	if err != nil || len(flags) == 0 {
		return err
	}
	notes := make([]string, 0, len(flags))
	for i := range flags {
		flags[i].ID = uuid.New()
		flags[i].TourAdvanceRequestID = tourID
		flags[i].CreatedAt = time.Now()
		var count int64
		if err := tx.Model(&models.TourAdvanceFlag{}).Where("tour_advance_request_id = ? AND kind = ?", tourID, flags[i].Kind).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := tx.Create(&flags[i]).Error; err != nil {
			return err
		}
		notes = append(notes, flags[i].Reason)
	}
	if len(notes) == 0 {
		return nil
	}
	return writeTourEvent(tx, tourID, models.TourEventFlagged, actor, strings.Join(notes, "; "))
}

func (s *TourAdvanceService) collectFlags(tx *gorm.DB, engineerID uuid.UUID, amount decimal.Decimal, additional bool) ([]models.TourAdvanceFlag, error) {
	var policy models.TourAdvancePolicy
	err := tx.First(&policy).Error
	threshold := (*decimal.Decimal)(nil)
	if err == nil {
		threshold = policy.HighValueThreshold
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	flags := make([]models.TourAdvanceFlag, 0)
	if threshold != nil && amount.GreaterThan(*threshold) {
		kind := "HIGH_VALUE"
		reason := fmt.Sprintf("Requested amount exceeds the review threshold of %s", threshold.StringFixed(2))
		if additional {
			kind = "ADDITIONAL_HIGH"
			reason = fmt.Sprintf("Additional amount exceeds the review threshold of %s", threshold.StringFixed(2))
		}
		flags = append(flags, models.TourAdvanceFlag{Kind: kind, Reason: reason})
	}
	if !additional {
		var prior []models.TourAdvanceRequest
		if err := tx.Select("approved_amount").Where("engineer_id = ? AND approved_amount IS NOT NULL", engineerID).Find(&prior).Error; err != nil {
			return nil, err
		}
		amounts := make([]decimal.Decimal, 0, len(prior))
		for _, row := range prior {
			if row.ApprovedAmount != nil {
				amounts = append(amounts, *row.ApprovedAmount)
			}
		}
		if domain.IsHistoricallyHigh(amount, amounts) {
			flags = append(flags, models.TourAdvanceFlag{Kind: "HISTORICAL", Reason: "Requested amount is more than twice this engineer's average approved advance"})
		}
		var nonBill int64
		if err := tx.Model(&models.TourAdvanceExpense{}).Where("engineer_id = ? AND bill_type = ?", engineerID, models.TourBillTypeNonBill).Count(&nonBill).Error; err != nil {
			return nil, err
		}
		if nonBill > 0 {
			flags = append(flags, models.TourAdvanceFlag{Kind: "NON_BILL_HISTORY", Reason: "Earlier tours include non-bill expenses"})
		}
	}
	return flags, nil
}

func originalFlagged(flags []models.TourAdvanceFlag) bool {
	for _, flag := range flags {
		switch flag.Kind {
		case "HIGH_VALUE", "HISTORICAL", "NON_BILL_HISTORY":
			return true
		}
	}
	return false
}

func additionalFlagged(flags []models.TourAdvanceFlag) bool {
	for _, flag := range flags {
		if flag.Kind == "ADDITIONAL_HIGH" {
			return true
		}
	}
	return false
}
