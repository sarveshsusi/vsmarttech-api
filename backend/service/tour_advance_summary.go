package service

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"rbac/domain"
	"rbac/models"
)

func (s *TourAdvanceService) FinancialSummary(actor uuid.UUID, role models.Role, filter TourSummaryFilter) (*TourFinancialSummary, error) {
	if role != models.RoleAdmin && role != models.RoleSuperAdmin {
		return nil, ErrTourForbidden
	}
	where, args, err := summaryWhere(filter)
	if err != nil {
		return nil, err
	}
	out := &TourFinancialSummary{
		TotalRequested: "0.00", OriginalApproved: "0.00", AdditionalRequested: "0.00", AdditionalApproved: "0.00",
		TotalDisbursed: "0.00", BillClaimed: "0.00", BillVerified: "0.00", NonBillClaimed: "0.00", NonBillVerified: "0.00",
		TotalVerified: "0.00", Outstanding: "0.00", CarryForward: []EngineerBalance{},
	}
	base := " FROM tour_advance_requests r WHERE " + where
	if err := scanMoney(s.db, "SELECT COALESCE(CAST(SUM(r.requested_amount) AS TEXT), '0')"+base, args, &out.TotalRequested); err != nil {
		return nil, err
	}
	if err := scanMoney(s.db, "SELECT COALESCE(CAST(SUM(r.approved_amount) AS TEXT), '0')"+base, args, &out.OriginalApproved); err != nil {
		return nil, err
	}
	if err := scanMoney(s.db, `SELECT COALESCE(CAST(SUM(a.amount) AS TEXT), '0') FROM tour_advance_additionals a
		JOIN tour_advance_requests r ON r.id = a.tour_advance_request_id WHERE `+where, args, &out.AdditionalRequested); err != nil {
		return nil, err
	}
	if err := scanMoney(s.db, `SELECT COALESCE(CAST(SUM(a.approved_amount) AS TEXT), '0') FROM tour_advance_additionals a
		JOIN tour_advance_requests r ON r.id = a.tour_advance_request_id WHERE a.status = 'APPROVED' AND `+where, args, &out.AdditionalApproved); err != nil {
		return nil, err
	}
	if err := scanMoney(s.db, `SELECT COALESCE(CAST(SUM(d.amount) AS TEXT), '0') FROM tour_advance_disbursements d
		JOIN tour_advance_requests r ON r.id = d.tour_advance_request_id WHERE `+where, args, &out.TotalDisbursed); err != nil {
		return nil, err
	}
	type expenseSums struct {
		BillClaimed, BillVerified, NonBillClaimed, NonBillVerified string
	}
	var sums expenseSums
	if err := s.db.Raw(`SELECT
		COALESCE(CAST(SUM(CASE WHEN e.bill_type = 'BILL' THEN e.claimed_amount ELSE 0 END) AS TEXT), '0'),
		COALESCE(CAST(SUM(CASE WHEN e.bill_type = 'BILL' AND e.status = 'VERIFIED' THEN e.accepted_amount ELSE 0 END) AS TEXT), '0'),
		COALESCE(CAST(SUM(CASE WHEN e.bill_type = 'NON_BILL' THEN e.claimed_amount ELSE 0 END) AS TEXT), '0'),
		COALESCE(CAST(SUM(CASE WHEN e.bill_type = 'NON_BILL' AND e.status = 'VERIFIED' THEN e.accepted_amount ELSE 0 END) AS TEXT), '0')
		FROM tour_advance_expenses e JOIN tour_advance_requests r ON r.id = e.tour_advance_request_id WHERE `+where, args...).Row().
		Scan(&sums.BillClaimed, &sums.BillVerified, &sums.NonBillClaimed, &sums.NonBillVerified); err != nil {
		return nil, err
	}
	out.BillClaimed = fixedMoney(sums.BillClaimed)
	out.BillVerified = fixedMoney(sums.BillVerified)
	out.NonBillClaimed = fixedMoney(sums.NonBillClaimed)
	out.NonBillVerified = fixedMoney(sums.NonBillVerified)
	billVerified, _ := decimal.NewFromString(out.BillVerified)
	nonBillVerified, _ := decimal.NewFromString(out.NonBillVerified)
	out.TotalVerified = billVerified.Add(nonBillVerified).StringFixed(2)
	openWhere := where + " AND r.status NOT IN ('REJECTED', 'PROCESSED')"
	if err := scanMoney(s.db, `SELECT COALESCE(CAST(SUM(
		COALESCE(r.credit_applied, 0)
		+ COALESCE((SELECT SUM(d.amount) FROM tour_advance_disbursements d WHERE d.tour_advance_request_id = r.id), 0)
		- COALESCE((SELECT SUM(e.accepted_amount) FROM tour_advance_expenses e WHERE e.tour_advance_request_id = r.id AND e.status = 'VERIFIED'), 0)
	) AS TEXT), '0') FROM tour_advance_requests r WHERE `+openWhere, args, &out.Outstanding); err != nil {
		return nil, err
	}
	if err := s.db.Raw("SELECT COUNT(*) FROM tour_advance_requests r WHERE "+openWhere, args...).Scan(&out.UnsettledTours).Error; err != nil {
		return nil, err
	}
	if err := s.db.Raw(`SELECT COUNT(*) FROM tour_advance_requests r WHERE `+where+`
		AND r.status = 'BILL_SUBMITTED' AND EXISTS (
			SELECT 1 FROM tour_advance_expenses e WHERE e.tour_advance_request_id = r.id AND e.status <> 'VERIFIED'
		)`, args...).Scan(&out.AwaitingVerification).Error; err != nil {
		return nil, err
	}
	if err := s.db.Raw(`SELECT COUNT(DISTINCT r.id) FROM tour_advance_requests r
		JOIN tour_advance_expenses e ON e.tour_advance_request_id = r.id
		WHERE `+where+` AND e.status IN ('REJECTED', 'NEEDS_CLARIFICATION')`, args...).Scan(&out.RejectedOrIncomplete).Error; err != nil {
		return nil, err
	}
	if err := s.db.Raw(`SELECT COUNT(DISTINCT r.id) FROM tour_advance_requests r
		JOIN tour_advance_flags f ON f.tour_advance_request_id = r.id
		WHERE `+where+` AND f.kind IN ('HIGH_VALUE', 'ADDITIONAL_HIGH', 'HISTORICAL')`, args...).Scan(&out.HighValueRequests).Error; err != nil {
		return nil, err
	}
	if err := s.db.Raw(`SELECT COUNT(DISTINCT r.engineer_id) FROM tour_advance_requests r
		WHERE `+where+` AND r.status NOT IN ('REJECTED', 'PROCESSED')`, args...).Scan(&out.BlockedEngineers).Error; err != nil {
		return nil, err
	}
	balances, err := s.carryForward(filter.EngineerID)
	if err != nil {
		return nil, err
	}
	out.CarryForward = balances
	return out, nil
}

func summaryWhere(filter TourSummaryFilter) (string, []interface{}, error) {
	parts := []string{"1 = 1"}
	args := make([]interface{}, 0)
	if id := strings.TrimSpace(filter.EngineerID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return "", nil, &domain.RuleError{Msg: "engineer is invalid"}
		}
		parts = append(parts, "r.engineer_id = ?")
		args = append(args, parsed)
	}
	if id := strings.TrimSpace(filter.CustomerID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return "", nil, &domain.RuleError{Msg: "customer is invalid"}
		}
		parts = append(parts, "r.customer_id = ?")
		args = append(args, parsed)
	}
	if id := strings.TrimSpace(filter.CompanyID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return "", nil, &domain.RuleError{Msg: "company is invalid"}
		}
		parts = append(parts, "r.customer_id IN (SELECT id FROM customers WHERE company_id = ?)")
		args = append(args, parsed)
	}
	if location := strings.TrimSpace(filter.Location); location != "" {
		parts = append(parts, "LOWER(r.location_snapshot) LIKE ?")
		args = append(args, "%"+strings.ToLower(escapeLike(location))+"%")
	}
	if number := strings.TrimSpace(filter.RequestNumber); number != "" {
		parts = append(parts, "LOWER(r.request_number) LIKE ?")
		args = append(args, "%"+strings.ToLower(escapeLike(number))+"%")
	}
	if filter.From != "" {
		from, err := time.Parse("2006-01-02", filter.From)
		if err != nil {
			return "", nil, &domain.RuleError{Msg: "from date is invalid"}
		}
		parts = append(parts, "r.created_at >= ?")
		args = append(args, from)
	}
	if filter.To != "" {
		to, err := time.Parse("2006-01-02", filter.To)
		if err != nil {
			return "", nil, &domain.RuleError{Msg: "to date is invalid"}
		}
		parts = append(parts, "r.created_at < ?")
		args = append(args, to.AddDate(0, 0, 1))
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		if !knownTourStatus(status) {
			return "", nil, &domain.RuleError{Msg: "invalid status"}
		}
		parts = append(parts, "r.status = ?")
		args = append(args, status)
	}
	switch strings.TrimSpace(filter.Verification) {
	case "":
	case "NOT_SUBMITTED":
		parts = append(parts, "NOT EXISTS (SELECT 1 FROM tour_advance_expenses e WHERE e.tour_advance_request_id = r.id)")
	case "PENDING_REVIEW", "NEEDS_CLARIFICATION", "VERIFIED", "REJECTED":
		parts = append(parts, "EXISTS (SELECT 1 FROM tour_advance_expenses e WHERE e.tour_advance_request_id = r.id AND e.status = ?)")
		args = append(args, filter.Verification)
	default:
		return "", nil, &domain.RuleError{Msg: "invalid verification status"}
	}
	return strings.Join(parts, " AND "), args, nil
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	return strings.ReplaceAll(value, "_", `\_`)
}

func scanMoney(db *gorm.DB, query string, args []interface{}, dest *string) error {
	var raw string
	if err := db.Raw(query, args...).Scan(&raw).Error; err != nil {
		return err
	}
	*dest = fixedMoney(raw)
	return nil
}

func fixedMoney(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "0.00"
	}
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		return "0.00"
	}
	return amount.StringFixed(2)
}

func (s *TourAdvanceService) carryForward(engineerID string) ([]EngineerBalance, error) {
	query := `SELECT l.engineer_id, COALESCE(u.name, ''),
		COALESCE(CAST(SUM(CASE WHEN l.entry_type = 'CREDIT_ISSUED' THEN l.amount ELSE 0 END)
		- SUM(CASE WHEN l.entry_type = 'CREDIT_APPLIED' THEN l.amount ELSE 0 END) AS TEXT), '0')
		FROM tour_advance_ledger l
		LEFT JOIN support_engineers e ON e.id = l.engineer_id
		LEFT JOIN users u ON u.id = e.user_id`
	args := make([]interface{}, 0)
	if id := strings.TrimSpace(engineerID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, &domain.RuleError{Msg: "engineer is invalid"}
		}
		query += " WHERE l.engineer_id = ?"
		args = append(args, parsed)
	}
	query += " GROUP BY l.engineer_id, u.name"
	rows, err := s.db.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]EngineerBalance, 0)
	for rows.Next() {
		var item EngineerBalance
		var raw string
		if err := rows.Scan(&item.EngineerID, &item.EngineerName, &raw); err != nil {
			return nil, err
		}
		item.Balance = fixedMoney(raw)
		amount, _ := decimal.NewFromString(item.Balance)
		if amount.GreaterThan(decimal.Zero) {
			out = append(out, item)
		}
	}
	return out, rows.Err()
}

func (s *TourAdvanceService) EngineerHistory(actor uuid.UUID, role models.Role, engineerID uuid.UUID) (*EngineerFinanceHistory, error) {
	if role != models.RoleAdmin && role != models.RoleSuperAdmin && role != models.RoleSupport {
		return nil, ErrTourForbidden
	}
	var engineer models.SupportEngineer
	if err := s.db.Preload("User").First(&engineer, "id = ?", engineerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTourNotFound
		}
		return nil, err
	}
	if role == models.RoleSupport && engineer.UserID != actor {
		return nil, ErrTourNotFound
	}
	_, pool, err := creditLots(s.db, engineerID)
	if err != nil {
		return nil, err
	}
	var tours []models.TourAdvanceRequest
	if err := s.withViewPreloads(s.db).Where("engineer_id = ?", engineerID).Order("created_at DESC").Limit(200).Find(&tours).Error; err != nil {
		return nil, err
	}
	views := make([]TourAdvanceView, 0, len(tours))
	for i := range tours {
		if role == models.RoleSupport && !s.canView(actor, role, &tours[i]) {
			continue
		}
		views = append(views, toTourView(&tours[i]))
	}
	var ledger []models.TourAdvanceLedgerEntry
	if err := s.db.Where("engineer_id = ?", engineerID).Order("created_at ASC").Find(&ledger).Error; err != nil {
		return nil, err
	}
	entries := make([]TourLedgerView, 0, len(ledger))
	for _, row := range ledger {
		item := TourLedgerView{
			EntryType: row.EntryType, Amount: row.Amount.StringFixed(2),
			BalanceBefore: row.BalanceBefore.StringFixed(2), BalanceAfter: row.BalanceAfter.StringFixed(2),
			OriginatingTourID: row.OriginatingTourID.String(), Reason: row.Reason, CreatedAt: row.CreatedAt,
		}
		if row.ConsumingTourID != nil {
			item.ConsumingTourID = row.ConsumingTourID.String()
		}
		entries = append(entries, item)
	}
	return &EngineerFinanceHistory{
		EngineerID: engineerID.String(), EngineerName: engineer.User.Name,
		AvailableCredit: pool.StringFixed(2), Tours: views, Ledger: entries,
	}, nil
}

func (s *TourAdvanceService) OwnHistory(actor uuid.UUID, role models.Role) (*EngineerFinanceHistory, error) {
	if role != models.RoleSupport {
		return nil, ErrTourForbidden
	}
	var engineer models.SupportEngineer
	if err := s.db.Where("user_id = ?", actor).First(&engineer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTourNotFound
		}
		return nil, err
	}
	return s.EngineerHistory(actor, role, engineer.ID)
}

func (s *TourAdvanceService) GetPolicy(actor uuid.UUID, role models.Role) (*TourPolicyView, error) {
	if role != models.RoleAdmin {
		return nil, ErrTourForbidden
	}
	var policy models.TourAdvancePolicy
	err := s.db.First(&policy).Error
	view := &TourPolicyView{}
	if err == gorm.ErrRecordNotFound {
		return view, nil
	}
	if err != nil {
		return nil, err
	}
	if policy.HighValueThreshold != nil {
		formatted := policy.HighValueThreshold.StringFixed(2)
		view.HighValueThreshold = &formatted
	}
	return view, nil
}

func (s *TourAdvanceService) UpdatePolicy(actor uuid.UUID, role models.Role, raw *string) (*TourPolicyView, error) {
	if role != models.RoleAdmin {
		return nil, ErrTourForbidden
	}
	var threshold *decimal.Decimal
	if raw != nil && strings.TrimSpace(*raw) != "" {
		amount, err := domain.ParseMoney(*raw)
		if err != nil {
			return nil, err
		}
		if !amount.GreaterThan(decimal.Zero) {
			return nil, &domain.RuleError{Msg: "threshold must be greater than 0"}
		}
		threshold = &amount
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var policy models.TourAdvancePolicy
		err := tx.First(&policy).Error
		now := time.Now()
		if err == gorm.ErrRecordNotFound {
			return tx.Create(&models.TourAdvancePolicy{
				ID: uuid.New(), HighValueThreshold: threshold, UpdatedBy: &actor, UpdatedAt: now, Singleton: true,
			}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&policy).Updates(map[string]interface{}{
			"high_value_threshold": threshold, "updated_by": actor, "updated_at": now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetPolicy(actor, role)
}
