package repository

import (
	"time"

	"rbac/models"

	"gorm.io/gorm"
)

type DashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) FetchAdminStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	var usersCount int64
	var ticketsCount int64
	var pendingTickets int64
	var closedTickets int64

	if err := r.db.
		Table("users").
		Where("is_active = true").
		Count(&usersCount).Error; err != nil {
		return nil, err
	}

	if err := r.db.
		Table("tickets").
		Count(&ticketsCount).Error; err != nil {
		return nil, err
	}

	// Pending = Open OR Assigned OR In Progress
	if err := r.db.
		Table("tickets").
		Where("status IN ?", []string{
			"Open",
			"Assigned",
			"In Progress",
		}).
		Count(&pendingTickets).Error; err != nil {
		return nil, err
	}

	// Closed = Closed status
	if err := r.db.
		Table("tickets").
		Where("status = ?", "Closed").
		Count(&closedTickets).Error; err != nil {
		return nil, err
	}

	stats["users"] = usersCount
	stats["tickets"] = ticketsCount
	stats["pending_tickets"] = pendingTickets
	stats["closed_tickets"] = closedTickets

	return stats, nil
}

// GetDashboardTickets returns tickets with filters (company, contract type, status, date range)
func (r *DashboardRepository) GetDashboardTickets(
	companyID *string,
	contractType *string,
	status *string,
	startDate *time.Time,
	endDate *time.Time,
	limit int,
	offset int,
) ([]models.Ticket, int64, error) {
	var tickets []models.Ticket
	var total int64

	// Build base query for counting (without preloads)
	countQuery := r.db.Model(&models.Ticket{})

	// Filter by company name
	if companyID != nil && *companyID != "" {
		countQuery = countQuery.
			Joins("JOIN customers ON tickets.customer_id = customers.id").
			Joins("JOIN companies ON customers.company_id = companies.id").
			Where("companies.name ILIKE ?", "%"+*companyID+"%")
	}

	// Filter by contract type (AMC, Warranty)
	if contractType != nil && *contractType != "" {
		countQuery = countQuery.
			Joins("JOIN customer_solutions ON tickets.customer_solution_id = customer_solutions.id").
			Where("customer_solutions.contract_type = ?", *contractType)
	}

	// Filter by status
	if status != nil && *status != "" {
		countQuery = countQuery.Where("tickets.status = ?", *status)
	}

	// Filter by date range
	if startDate != nil {
		countQuery = countQuery.Where("tickets.created_at >= ?", startDate)
	}
	if endDate != nil {
		countQuery = countQuery.Where("tickets.created_at <= ?", endDate)
	}

	// Count total
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Build fetch query with preloads
	fetchQuery := r.db.
		Preload("Customer", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Company")
		}).
		Preload("CustomerSolution").
		Preload("CustomerSolution.Solution")

	// Apply same filters for fetch
	if companyID != nil && *companyID != "" {
		fetchQuery = fetchQuery.
			Joins("JOIN customers ON tickets.customer_id = customers.id").
			Joins("JOIN companies ON customers.company_id = companies.id").
			Where("companies.name ILIKE ?", "%"+*companyID+"%")
	}

	if contractType != nil && *contractType != "" {
		fetchQuery = fetchQuery.
			Joins("JOIN customer_solutions ON tickets.customer_solution_id = customer_solutions.id").
			Where("customer_solutions.contract_type = ?", *contractType)
	}

	if status != nil && *status != "" {
		fetchQuery = fetchQuery.Where("tickets.status = ?", *status)
	}

	if startDate != nil {
		fetchQuery = fetchQuery.Where("tickets.created_at >= ?", startDate)
	}
	if endDate != nil {
		fetchQuery = fetchQuery.Where("tickets.created_at <= ?", endDate)
	}

	// Fetch with pagination
	if err := fetchQuery.
		Order("tickets.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

// ========================
// SUPPORT DASHBOARD STATS
// ========================

type SupportStats struct {
	TotalTickets   int64 `json:"total_tickets"`
	ClosedTickets  int64 `json:"closed_tickets"`
	PendingTickets int64 `json:"pending_tickets"`
	OpenTickets    int64 `json:"open_tickets"`
}

func (r *DashboardRepository) GetSupportStats() (*SupportStats, error) {
	stats := &SupportStats{}

	// Total tickets
	if err := r.db.
		Table("tickets").
		Count(&stats.TotalTickets).Error; err != nil {
		return nil, err
	}

	// Closed tickets
	if err := r.db.
		Table("tickets").
		Where("status = ?", "Closed").
		Count(&stats.ClosedTickets).Error; err != nil {
		return nil, err
	}

	// Pending tickets (Open, Assigned, In Progress)
	if err := r.db.
		Table("tickets").
		Where("status IN ?", []string{"Open", "Assigned", "In Progress"}).
		Count(&stats.PendingTickets).Error; err != nil {
		return nil, err
	}

	// Open tickets (not closed or assigned)
	if err := r.db.
		Table("tickets").
		Where("status = ?", "Open").
		Count(&stats.OpenTickets).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// ========================
// CUSTOMER DASHBOARD STATS
// ========================

type CustomerStats struct {
	TotalTickets  int64 `json:"total_tickets"`
	OpenTickets   int64 `json:"open_tickets"`
	ClosedTickets int64 `json:"closed_tickets"`
	InProgress    int64 `json:"in_progress"`
}

type TicketCheckpoint struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *DashboardRepository) GetCustomerStats(customerID string) (*CustomerStats, error) {
	stats := &CustomerStats{}

	// Total tickets for this customer
	if err := r.db.
		Table("tickets").
		Where("customer_id = ?", customerID).
		Count(&stats.TotalTickets).Error; err != nil {
		return nil, err
	}

	// Open tickets
	if err := r.db.
		Table("tickets").
		Where("customer_id = ? AND status = ?", customerID, "Open").
		Count(&stats.OpenTickets).Error; err != nil {
		return nil, err
	}

	// Closed tickets
	if err := r.db.
		Table("tickets").
		Where("customer_id = ? AND status = ?", customerID, "Closed").
		Count(&stats.ClosedTickets).Error; err != nil {
		return nil, err
	}

	// In Progress tickets
	if err := r.db.
		Table("tickets").
		Where("customer_id = ? AND status = ?", customerID, "In Progress").
		Count(&stats.InProgress).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetCustomerTickets returns all tickets for a customer with status tracking
func (r *DashboardRepository) GetCustomerTickets(customerID string) ([]TicketCheckpoint, error) {
	var tickets []TicketCheckpoint

	if err := r.db.
		Table("tickets").
		Select("id, title, status, created_at, updated_at").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&tickets).Error; err != nil {
		return nil, err
	}

	// Return empty slice instead of nil to ensure JSON serializes as [] not null
	if tickets == nil {
		tickets = []TicketCheckpoint{}
	}
	return tickets, nil
}

// OpsKPIs holds operational metrics for the admin dashboard.
type OpsKPIs struct {
	WindowDays           int     `json:"window_days"`
	MTTRHours            float64 `json:"mttr_hours"`
	FirstResponseHours   float64 `json:"first_response_hours"`
	ReopenRatePct        float64 `json:"reopen_rate_pct"`
	VisitsPerTicket      float64 `json:"visits_per_ticket"`
	SLAAttainmentPct     float64 `json:"sla_attainment_pct"`
	OpenBreached         int64   `json:"open_breached"`
	CSATAverage          float64 `json:"csat_average"`
	CSATCount            int64   `json:"csat_count"`
	ClosedInWindow       int64   `json:"closed_in_window"`
	TicketsWithVisits    int64   `json:"tickets_with_visits"`
}

func (r *DashboardRepository) FetchOpsKPIs(windowDays int) (*OpsKPIs, error) {
	if windowDays <= 0 {
		windowDays = 90
	}
	since := time.Now().AddDate(0, 0, -windowDays)
	out := &OpsKPIs{WindowDays: windowDays}

	// MTTR: avg hours from created → closed for closed tickets in window
	var mttr *float64
	_ = r.db.Raw(`
		SELECT AVG(EXTRACT(EPOCH FROM (closed_at - created_at)) / 3600.0)
		FROM tickets
		WHERE status = 'Closed'
		  AND closed_at IS NOT NULL
		  AND closed_at >= ?
	`, since).Scan(&mttr).Error
	if mttr != nil {
		out.MTTRHours = round2(*mttr)
	}

	_ = r.db.Raw(`
		SELECT COUNT(*) FROM tickets
		WHERE status = 'Closed' AND closed_at IS NOT NULL AND closed_at >= ?
	`, since).Scan(&out.ClosedInWindow).Error

	// First response: first assigned/started event after create
	var frt *float64
	_ = r.db.Raw(`
		SELECT AVG(EXTRACT(EPOCH FROM (fr.first_at - t.created_at)) / 3600.0)
		FROM tickets t
		INNER JOIN (
			SELECT ticket_id, MIN(created_at) AS first_at
			FROM ticket_events
			WHERE event_type IN ('assigned', 'started')
			GROUP BY ticket_id
		) fr ON fr.ticket_id = t.id
		WHERE t.created_at >= ?
	`, since).Scan(&frt).Error
	if frt != nil {
		out.FirstResponseHours = round2(*frt)
	}

	// Reopen rate: tickets with ≥1 reopen event / closed tickets in window
	var reopened int64
	_ = r.db.Raw(`
		SELECT COUNT(DISTINCT t.id)
		FROM tickets t
		INNER JOIN ticket_events e ON e.ticket_id = t.id AND e.event_type = 'reopened'
		WHERE t.status = 'Closed'
		  AND t.closed_at IS NOT NULL
		  AND t.closed_at >= ?
	`, since).Scan(&reopened).Error
	if out.ClosedInWindow > 0 {
		out.ReopenRatePct = round2(float64(reopened) * 100.0 / float64(out.ClosedInWindow))
	}

	// Visits per ticket (among tickets that have ≥1 visit, created in window)
	var vpt *float64
	_ = r.db.Raw(`
		SELECT AVG(cnt)::float FROM (
			SELECT ticket_id, COUNT(*) AS cnt
			FROM service_visits
			GROUP BY ticket_id
		) v
		INNER JOIN tickets t ON t.id = v.ticket_id
		WHERE t.created_at >= ?
	`, since).Scan(&vpt).Error
	if vpt != nil {
		out.VisitsPerTicket = round2(*vpt)
	}
	_ = r.db.Raw(`
		SELECT COUNT(DISTINCT sv.ticket_id)
		FROM service_visits sv
		INNER JOIN tickets t ON t.id = sv.ticket_id
		WHERE t.created_at >= ?
	`, since).Scan(&out.TicketsWithVisits).Error

	// SLA attainment: closed with closed_at <= target_at
	var slaMet, slaTotal int64
	_ = r.db.Raw(`
		SELECT COUNT(*) FROM tickets
		WHERE status = 'Closed'
		  AND closed_at IS NOT NULL
		  AND target_at IS NOT NULL
		  AND closed_at >= ?
		  AND closed_at <= target_at
	`, since).Scan(&slaMet).Error
	_ = r.db.Raw(`
		SELECT COUNT(*) FROM tickets
		WHERE status = 'Closed'
		  AND closed_at IS NOT NULL
		  AND target_at IS NOT NULL
		  AND closed_at >= ?
	`, since).Scan(&slaTotal).Error
	if slaTotal > 0 {
		out.SLAAttainmentPct = round2(float64(slaMet) * 100.0 / float64(slaTotal))
	}

	_ = r.db.Raw(`
		SELECT COUNT(*) FROM tickets
		WHERE status != 'Closed'
		  AND target_at IS NOT NULL
		  AND target_at < NOW()
	`).Scan(&out.OpenBreached).Error

	// CSAT
	var csat *float64
	_ = r.db.Raw(`
		SELECT AVG(rating)::float FROM ticket_feedbacks
		WHERE created_at >= ?
	`, since).Scan(&csat).Error
	if csat != nil {
		out.CSATAverage = round2(*csat)
	}
	_ = r.db.Raw(`
		SELECT COUNT(*) FROM ticket_feedbacks WHERE created_at >= ?
	`, since).Scan(&out.CSATCount).Error

	return out, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
