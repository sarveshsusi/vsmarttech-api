package service

import (
	"errors"
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

func (s *TourAdvanceService) SubmitExpenses(actor uuid.UUID, role models.Role, id uuid.UUID, lines []ExpenseLineInput) (*TourAdvanceView, error) {
	if role != models.RoleAdmin {
		return nil, ErrTourForbidden
	}
	if len(lines) == 0 {
		return nil, &domain.RuleError{Msg: "add at least one expense"}
	}
	if len(lines) > 40 {
		return nil, &domain.RuleError{Msg: "too many expense lines"}
	}
	parsed := make([]models.TourAdvanceExpense, 0, len(lines))
	for _, line := range lines {
		item, err := parseExpenseLine(line)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, item)
	}
	var becameReady bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		q := tx
		if tx.Dialector.Name() == "postgres" {
			q = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var current models.TourAdvanceRequest
		if err := q.First(&current, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTourNotFound
			}
			return err
		}
		if current.Status != models.TourStatusApproved && current.Status != models.TourStatusBillSubmitted {
			return ErrInvalidTransition
		}
		var existing int64
		if err := tx.Model(&models.TourAdvanceExpense{}).Where("tour_advance_request_id = ?", id).Count(&existing).Error; err != nil {
			return err
		}
		if existing+int64(len(parsed)) > 40 {
			return &domain.RuleError{Msg: "too many expense lines"}
		}
		now := time.Now()
		for i := range parsed {
			accepted := new(decimal.Decimal)
			*accepted = parsed[i].ClaimedAmount
			reviewedBy := actor
			reviewedAt := now
			parsed[i].ID = uuid.New()
			parsed[i].TourAdvanceRequestID = id
			parsed[i].EngineerID = current.EngineerID
			parsed[i].Status = models.TourExpenseVerified
			parsed[i].AcceptedAmount = accepted
			parsed[i].ReviewedBy = &reviewedBy
			parsed[i].ReviewedAt = &reviewedAt
			parsed[i].CreatedAt = now
			parsed[i].UpdatedAt = now
		}
		if err := tx.Create(&parsed).Error; err != nil {
			return err
		}
		if current.Status == models.TourStatusApproved {
			res := tx.Model(&models.TourAdvanceRequest{}).Where("id = ? AND status = ?", id, models.TourStatusApproved).Updates(map[string]interface{}{
				"status": models.TourStatusBillSubmitted, "bill_submitted_by": actor, "bill_submitted_at": now, "updated_at": now,
			})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrInvalidTransition
			}
			if err := writeTourEvent(tx, id, models.TourEventBillSubmitted, actor, fmt.Sprintf("%d expense lines", len(parsed))); err != nil {
				return err
			}
		} else if err := writeTourEvent(tx, id, models.TourEventExpenseVerified, actor, fmt.Sprintf("%d expense lines", len(parsed))); err != nil {
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
	if becameReady {
		s.notifyRole(models.RoleAdmin, models.NotificationTypeTourAdvanceReady, "Tour ready to process",
			fmt.Sprintf("%s is ready for final processing.", view.RequestNumber), &models.TourAdvanceRequest{ID: id, RequestNumber: view.RequestNumber})
	}
	return view, nil
}

func (s *TourAdvanceService) AmendExpense(actor uuid.UUID, role models.Role, tourID, expenseID uuid.UUID, line ExpenseLineInput) (*TourAdvanceView, error) {
	if role != models.RoleAdmin {
		return nil, ErrTourForbidden
	}
	item, err := parseExpenseLine(line)
	if err != nil {
		return nil, err
	}
	var becameReady bool
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var current models.TourAdvanceExpense
		if err := tx.Where("id = ? AND tour_advance_request_id = ?", expenseID, tourID).First(&current).Error; err != nil {
			return ErrTourNotFound
		}
		var tour models.TourAdvanceRequest
		if err := tx.First(&tour, "id = ?", tourID).Error; err != nil {
			return ErrTourNotFound
		}
		if tour.Status != models.TourStatusBillSubmitted {
			return ErrInvalidTransition
		}
		before := current.ClaimedAmount.StringFixed(2)
		now := time.Now()
		accepted := item.ClaimedAmount
		res := tx.Model(&models.TourAdvanceExpense{}).Where("id = ?", expenseID).Updates(map[string]interface{}{
			"category": item.Category, "bill_type": item.BillType, "expense_date": item.ExpenseDate,
			"description": item.Description, "claimed_amount": item.ClaimedAmount,
			"reference_number": item.ReferenceNumber, "missing_reason": item.MissingReason,
			"accepted_amount": accepted, "status": models.TourExpenseVerified, "review_reason": "",
			"reviewed_by": actor, "reviewed_at": now, "updated_at": now,
		})
		if res.Error != nil {
			return res.Error
		}
		if err := writeTourEvent(tx, tourID, models.TourEventExpenseAmended, actor, fmt.Sprintf("before %s after %s", before, item.ClaimedAmount.StringFixed(2))); err != nil {
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
	if becameReady {
		s.notifyRole(models.RoleAdmin, models.NotificationTypeTourAdvanceReady, "Tour ready to process",
			fmt.Sprintf("%s is ready for final processing.", view.RequestNumber), &models.TourAdvanceRequest{ID: tourID, RequestNumber: view.RequestNumber})
	}
	return view, nil
}

func parseExpenseLine(line ExpenseLineInput) (models.TourAdvanceExpense, error) {
	claimed, err := domain.ParseMoney(line.ClaimedAmount)
	if err != nil {
		return models.TourAdvanceExpense{}, err
	}
	rules := domain.ExpenseLineRules{
		Category: strings.ToUpper(strings.TrimSpace(line.Category)), BillType: strings.ToUpper(strings.TrimSpace(line.BillType)),
		ExpenseDate: strings.TrimSpace(line.ExpenseDate), Description: strings.TrimSpace(line.Description),
		Claimed: claimed, Reference: line.ReferenceNumber, Reason: line.MissingReason,
	}
	if err := domain.ValidateExpenseLine(rules); err != nil {
		return models.TourAdvanceExpense{}, err
	}
	return models.TourAdvanceExpense{
		Category: models.TourExpenseCategory(rules.Category), BillType: models.TourBillType(rules.BillType),
		ExpenseDate: rules.ExpenseDate, Description: rules.Description, ClaimedAmount: claimed,
		ReferenceNumber: strings.TrimSpace(rules.Reference), MissingReason: strings.TrimSpace(rules.Reason),
	}, nil
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
