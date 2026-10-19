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

func (s *TourAdvanceService) CreateAdditional(actor uuid.UUID, role models.Role, id uuid.UUID, amountRaw, reason string) (*TourAdvanceView, error) {
	if role != models.RoleSupport {
		return nil, ErrTourForbidden
	}
	row, err := s.load(id)
	if err != nil {
		return nil, err
	}
	if row.Engineer == nil || row.Engineer.UserID != actor {
		return nil, ErrTourNotFound
	}
	if row.Status == models.TourStatusRejected || row.Status == models.TourStatusProcessed {
		return nil, &domain.RuleError{Msg: "this tour cannot take an additional request"}
	}
	if row.Additional != nil {
		return nil, &domain.RuleError{Msg: "an additional request already exists"}
	}

	amount, err := domain.ParseMoney(amountRaw)
	if err != nil {
		return nil, err
	}
	if !amount.GreaterThan(decimal.Zero) {
		return nil, &domain.RuleError{Msg: "amount must be greater than 0"}
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, &domain.RuleError{Msg: "reason is required"}
	}
	if len(reason) > 2000 {
		return nil, &domain.RuleError{Msg: "reason is too long"}
	}

	now := time.Now()
	extra := models.TourAdvanceAdditional{
		ID:                   uuid.New(),
		TourAdvanceRequestID: id,
		Amount:               amount,
		Reason:               reason,
		Status:               models.TourAdditionalPending,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&extra).Error; err != nil {
			if isUniqueViolation(err) {
				return &domain.RuleError{Msg: "an additional request already exists"}
			}
			return err
		}
		return writeTourEvent(tx, id, models.TourEventAdditionalRequested, actor, reason)
	})
	if err != nil {
		return nil, err
	}

	var actorUser models.User
	_ = s.db.Select("name").First(&actorUser, "id = ?", actor).Error
	s.notifyRole(
		models.RoleSuperAdmin,
		models.NotificationTypeTourAdditionalSubmitted,
		"Additional tour money requested",
		fmt.Sprintf("%s requested additional money on %s.", actorUser.Name, row.RequestNumber),
		row,
	)
	return s.Get(actor, role, id)
}

func (s *TourAdvanceService) ApproveAdditional(actor uuid.UUID, role models.Role, id uuid.UUID, amountRaw, remarks, paymentMode, transactionID string) (*TourAdvanceView, error) {
	if role != models.RoleSuperAdmin {
		return nil, ErrTourForbidden
	}
	amount, err := domain.ParseMoney(amountRaw)
	if err != nil {
		return nil, err
	}
	remarks = strings.TrimSpace(remarks)
	if len(remarks) > 2000 {
		return nil, &domain.RuleError{Msg: "remarks are too long"}
	}
	paymentMode, transactionID, err = domain.NormalizePayment(paymentMode, transactionID)
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		extra, err := lockAdditional(tx, id)
		if err != nil {
			return err
		}
		if extra.Status != models.TourAdditionalPending {
			return ErrInvalidTransition
		}
		if err := domain.ValidateApprovedAmount(extra.Amount, amount); err != nil {
			return err
		}
		now := time.Now()
		updates := map[string]interface{}{
			"status":           models.TourAdditionalApproved,
			"approved_amount":  amount,
			"approval_remarks": remarks,
			"approved_by":      actor,
			"approved_at":      now,
			"payment_mode":     paymentMode,
			"updated_at":       now,
		}
		if transactionID == "" {
			updates["transaction_id"] = nil
		} else {
			updates["transaction_id"] = transactionID
		}
		res := tx.Model(&models.TourAdvanceAdditional{}).
			Where("id = ? AND status = ?", extra.ID, models.TourAdditionalPending).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidTransition
		}
		return writeTourEvent(tx, id, models.TourEventAdditionalApproved, actor, remarks)
	})
	if err != nil {
		return nil, err
	}

	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyEngineer(id, models.NotificationTypeTourAdditionalApproved, "Additional tour money approved",
		fmt.Sprintf("Additional money on %s was approved.", view.RequestNumber))
	return view, nil
}

func (s *TourAdvanceService) RejectAdditional(actor uuid.UUID, role models.Role, id uuid.UUID, remarks string) (*TourAdvanceView, error) {
	if role != models.RoleSuperAdmin {
		return nil, ErrTourForbidden
	}
	remarks = strings.TrimSpace(remarks)
	if remarks == "" {
		return nil, &domain.RuleError{Msg: "rejection reason is required"}
	}
	if len(remarks) > 2000 {
		return nil, &domain.RuleError{Msg: "remarks are too long"}
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		extra, err := lockAdditional(tx, id)
		if err != nil {
			return err
		}
		if extra.Status != models.TourAdditionalPending {
			return ErrInvalidTransition
		}
		now := time.Now()
		res := tx.Model(&models.TourAdvanceAdditional{}).
			Where("id = ? AND status = ?", extra.ID, models.TourAdditionalPending).
			Updates(map[string]interface{}{
				"status":            models.TourAdditionalRejected,
				"rejection_remarks": remarks,
				"rejected_by":       actor,
				"rejected_at":       now,
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidTransition
		}
		return writeTourEvent(tx, id, models.TourEventAdditionalRejected, actor, remarks)
	})
	if err != nil {
		return nil, err
	}

	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyEngineer(id, models.NotificationTypeTourAdditionalRejected, "Additional tour money rejected",
		fmt.Sprintf("Additional money on %s was rejected.", view.RequestNumber))
	return view, nil
}

func lockAdditional(tx *gorm.DB, requestID uuid.UUID) (*models.TourAdvanceAdditional, error) {
	q := tx
	if tx.Dialector.Name() == "postgres" {
		q = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var extra models.TourAdvanceAdditional
	err := q.Where("tour_advance_request_id = ?", requestID).First(&extra).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidTransition
	}
	if err != nil {
		return nil, err
	}
	return &extra, nil
}

func toAdditionalView(row *models.TourAdvanceAdditional) *TourAdditionalView {
	if row == nil {
		return nil
	}
	view := &TourAdditionalView{
		Amount:           row.Amount.StringFixed(2),
		Reason:           row.Reason,
		Status:           string(row.Status),
		ApprovalRemarks:  row.ApprovalRemarks,
		ApprovedByName:   userName(row.ApprovedByUser),
		ApprovedAt:       row.ApprovedAt,
		RejectionRemarks: row.RejectionRemarks,
		RejectedByName:   userName(row.RejectedByUser),
		RejectedAt:       row.RejectedAt,
		CreatedAt:        row.CreatedAt,
	}
	if row.ApprovedAmount != nil {
		formatted := row.ApprovedAmount.StringFixed(2)
		view.ApprovedAmount = &formatted
	}
	if row.PaymentMode != nil {
		view.PaymentMode = *row.PaymentMode
	}
	if row.TransactionID != nil {
		view.TransactionID = *row.TransactionID
	}
	return view
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}
