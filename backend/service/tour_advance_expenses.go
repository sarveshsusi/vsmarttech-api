package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/domain"
	"rbac/models"
)

func (s *TourAdvanceService) SubmitExpenses(actor uuid.UUID, role models.Role, id uuid.UUID, lines []ExpenseLineInput) (*TourAdvanceView, error) {
	if role != models.RoleSupport {
		return nil, ErrTourForbidden
	}
	if len(lines) == 0 {
		return nil, &domain.RuleError{Msg: "add at least one expense"}
	}
	if len(lines) > 40 {
		return nil, &domain.RuleError{Msg: "too many expense lines"}
	}
	row, err := s.load(id)
	if err != nil {
		return nil, err
	}
	if row.Engineer == nil || row.Engineer.UserID != actor {
		return nil, ErrTourNotFound
	}
	parsed := make([]models.TourAdvanceExpense, 0, len(lines))
	for _, line := range lines {
		claimed, err := domain.ParseMoney(line.ClaimedAmount)
		if err != nil {
			return nil, err
		}
		rules := domain.ExpenseLineRules{
			Category: strings.ToUpper(strings.TrimSpace(line.Category)), BillType: strings.ToUpper(strings.TrimSpace(line.BillType)),
			ExpenseDate: strings.TrimSpace(line.ExpenseDate), Description: strings.TrimSpace(line.Description),
			Claimed: claimed, Reference: line.ReferenceNumber, Reason: line.MissingReason,
		}
		if err := domain.ValidateExpenseLine(rules); err != nil {
			return nil, err
		}
		now := time.Now()
		parsed = append(parsed, models.TourAdvanceExpense{
			ID: uuid.New(), TourAdvanceRequestID: id, EngineerID: row.EngineerID,
			Category: models.TourExpenseCategory(rules.Category), BillType: models.TourBillType(rules.BillType),
			ExpenseDate: rules.ExpenseDate, Description: rules.Description, ClaimedAmount: claimed,
			ReferenceNumber: strings.TrimSpace(rules.Reference), MissingReason: strings.TrimSpace(rules.Reason),
			Status: models.TourExpensePendingReview, CreatedAt: now, UpdatedAt: now,
		})
	}
	err = s.transition(id, models.TourStatusApproved, models.TourStatusBillSubmitted, func(tx *gorm.DB, current *models.TourAdvanceRequest) error {
		now := time.Now()
		res := tx.Model(&models.TourAdvanceRequest{}).Where("id = ? AND status = ?", id, models.TourStatusApproved).Updates(map[string]interface{}{
			"status": models.TourStatusBillSubmitted, "bill_submitted_by": actor, "bill_submitted_at": now, "updated_at": now,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidTransition
		}
		if err := tx.Create(&parsed).Error; err != nil {
			return err
		}
		return writeTourEvent(tx, id, models.TourEventBillSubmitted, actor, fmt.Sprintf("%d expense lines", len(parsed)))
	})
	if err != nil {
		return nil, err
	}
	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyRole(models.RoleAdmin, models.NotificationTypeTourAdvanceBillSubmitted, "Expenses submitted",
		fmt.Sprintf("%s expenses were submitted for review.", view.RequestNumber), row)
	return view, nil
}

func (s *TourAdvanceService) AmendExpense(actor uuid.UUID, role models.Role, tourID, expenseID uuid.UUID, line ExpenseLineInput) (*TourAdvanceView, error) {
	if role != models.RoleSupport {
		return nil, ErrTourForbidden
	}
	claimed, err := domain.ParseMoney(line.ClaimedAmount)
	if err != nil {
		return nil, err
	}
	rules := domain.ExpenseLineRules{
		Category: strings.ToUpper(strings.TrimSpace(line.Category)), BillType: strings.ToUpper(strings.TrimSpace(line.BillType)),
		ExpenseDate: strings.TrimSpace(line.ExpenseDate), Description: strings.TrimSpace(line.Description),
		Claimed: claimed, Reference: line.ReferenceNumber, Reason: line.MissingReason,
	}
	if err := domain.ValidateExpenseLine(rules); err != nil {
		return nil, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var current models.TourAdvanceExpense
		if err := tx.Where("id = ? AND tour_advance_request_id = ?", expenseID, tourID).First(&current).Error; err != nil {
			return ErrTourNotFound
		}
		var tour models.TourAdvanceRequest
		if err := tx.Preload("Engineer").First(&tour, "id = ?", tourID).Error; err != nil {
			return ErrTourNotFound
		}
		if tour.Engineer == nil || tour.Engineer.UserID != actor {
			return ErrTourNotFound
		}
		if tour.Status != models.TourStatusBillSubmitted {
			return ErrInvalidTransition
		}
		if current.Status == models.TourExpenseVerified {
			return &domain.RuleError{Msg: "verified expenses cannot be changed"}
		}
		if current.Status != models.TourExpenseNeedsClarification && current.Status != models.TourExpenseRejected {
			return &domain.RuleError{Msg: "only a rejected or clarification line can be corrected"}
		}
		before := current.ClaimedAmount.StringFixed(2)
		now := time.Now()
		res := tx.Model(&models.TourAdvanceExpense{}).Where("id = ?", expenseID).Updates(map[string]interface{}{
			"category": rules.Category, "bill_type": rules.BillType, "expense_date": rules.ExpenseDate,
			"description": rules.Description, "claimed_amount": claimed,
			"reference_number": strings.TrimSpace(rules.Reference), "missing_reason": strings.TrimSpace(rules.Reason),
			"accepted_amount": nil, "status": models.TourExpensePendingReview, "review_reason": "",
			"reviewed_by": nil, "reviewed_at": nil, "updated_at": now,
		})
		if res.Error != nil {
			return res.Error
		}
		return writeTourEvent(tx, tourID, models.TourEventExpenseAmended, actor, fmt.Sprintf("before %s after %s", before, claimed.StringFixed(2)))
	})
	if err != nil {
		return nil, err
	}
	return s.Get(actor, role, tourID)
}

func (s *TourAdvanceService) ReviewExpense(actor uuid.UUID, role models.Role, tourID, expenseID uuid.UUID, decision, acceptedRaw, reason string) (*TourAdvanceView, error) {
	if role != models.RoleAdmin {
		return nil, ErrTourForbidden
	}
	decision = strings.ToUpper(strings.TrimSpace(decision))
	reason = strings.TrimSpace(reason)
	if len(reason) > 2000 {
		return nil, &domain.RuleError{Msg: "reason is too long"}
	}
	var becameReady bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var tour models.TourAdvanceRequest
		if err := tx.First(&tour, "id = ?", tourID).Error; err != nil {
			return ErrTourNotFound
		}
		if tour.Status != models.TourStatusBillSubmitted {
			return ErrInvalidTransition
		}
		var line models.TourAdvanceExpense
		if err := tx.Where("id = ? AND tour_advance_request_id = ?", expenseID, tourID).First(&line).Error; err != nil {
			return ErrTourNotFound
		}
		if line.Status == models.TourExpenseVerified {
			return &domain.RuleError{Msg: "verified expenses cannot be changed"}
		}
		now := time.Now()
		updates := map[string]interface{}{"reviewed_by": actor, "reviewed_at": now, "updated_at": now, "review_reason": reason}
		action := models.TourEventExpenseClarified
		note := reason
		switch decision {
		case "ACCEPT":
			accepted, err := domain.ParseMoney(acceptedRaw)
			if err != nil {
				return err
			}
			if err := domain.ValidateAcceptedAmount(line.ClaimedAmount, accepted); err != nil {
				return err
			}
			updates["status"] = models.TourExpenseVerified
			updates["accepted_amount"] = accepted
			action = models.TourEventExpenseVerified
			note = fmt.Sprintf("accepted %s", accepted.StringFixed(2))
			if reason != "" {
				note += "; " + reason
			}
		case "REJECT":
			if reason == "" {
				return &domain.RuleError{Msg: "a reason is required"}
			}
			updates["status"] = models.TourExpenseRejected
			updates["accepted_amount"] = nil
			action = models.TourEventExpenseRejected
		case "CLARIFY":
			if reason == "" {
				return &domain.RuleError{Msg: "a reason is required"}
			}
			updates["status"] = models.TourExpenseNeedsClarification
			updates["accepted_amount"] = nil
		default:
			return &domain.RuleError{Msg: "choose accept, reject, or clarify"}
		}
		if err := tx.Model(&models.TourAdvanceExpense{}).Where("id = ?", expenseID).Updates(updates).Error; err != nil {
			return err
		}
		if err := writeTourEvent(tx, tourID, action, actor, note); err != nil {
			return err
		}
		ready, err := settlementReady(tx, tourID)
		if err != nil {
			return err
		}
		becameReady = ready
		if ready {
			return markReady(tx, tourID, actor)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	view, err := s.Get(actor, role, tourID)
	if err != nil {
		return nil, err
	}
	switch decision {
	case "REJECT":
		s.notifyEngineer(tourID, models.NotificationTypeTourExpenseRejected, "Expense rejected",
			fmt.Sprintf("An expense on %s was rejected.", view.RequestNumber))
	case "CLARIFY":
		s.notifyEngineer(tourID, models.NotificationTypeTourExpenseClarification, "Expense needs clarification",
			fmt.Sprintf("An expense on %s needs clarification.", view.RequestNumber))
	}
	if becameReady {
		s.notifyRole(models.RoleAdmin, models.NotificationTypeTourAdvanceReady, "Tour ready to process",
			fmt.Sprintf("%s is ready for final processing.", view.RequestNumber), &models.TourAdvanceRequest{ID: tourID, RequestNumber: view.RequestNumber})
	}
	return view, nil
}
