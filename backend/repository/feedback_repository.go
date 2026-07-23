package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
)

type FeedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Create(feedback *models.TicketFeedback) error {
	return r.db.Create(feedback).Error
}

func (r *FeedbackRepository) Update(feedback *models.TicketFeedback) error {
	return r.db.Save(feedback).Error
}

func (r *FeedbackRepository) UpdateFields(id uuid.UUID, fields map[string]interface{}) error {
	return r.db.Model(&models.TicketFeedback{}).Where("id = ?", id).Updates(fields).Error
}

func (r *FeedbackRepository) GetByID(id uuid.UUID) (*models.TicketFeedback, error) {
	var fb models.TicketFeedback
	err := r.db.
		Preload("Engineer.User").
		Preload("Customer").
		Preload("Company").
		Where("id = ?", id).
		First(&fb).Error
	if err != nil {
		return nil, err
	}
	return &fb, nil
}

func (r *FeedbackRepository) GetByTicketID(ticketID string) (*models.TicketFeedback, error) {
	var fb models.TicketFeedback
	err := r.db.
		Preload("Engineer.User").
		Preload("Customer").
		Preload("Company").
		Where("ticket_id = ?", ticketID).
		First(&fb).Error
	if err != nil {
		return nil, err
	}
	return &fb, nil
}

func (r *FeedbackRepository) GetByTicketIDs(ticketIDs []string) (map[string]models.TicketFeedback, error) {
	out := make(map[string]models.TicketFeedback)
	if len(ticketIDs) == 0 {
		return out, nil
	}
	var rows []models.TicketFeedback
	if err := r.db.Where("ticket_id IN ?", ticketIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TicketID] = row
	}
	return out, nil
}

type FeedbackListFilter struct {
	EngineerID      *uuid.UUID
	CustomerID      *uuid.UUID
	CompanyID       *uuid.UUID
	Status          *models.FeedbackStatus
	Rating          *int
	ServiceCallType *string
	Priority        *string
	From            *time.Time
	To              *time.Time
	Limit           int
	Offset          int
}

func (r *FeedbackRepository) applyFilters(q *gorm.DB, filter FeedbackListFilter) *gorm.DB {
	if filter.EngineerID != nil {
		q = q.Where("ticket_feedbacks.engineer_id = ?", *filter.EngineerID)
	}
	if filter.CustomerID != nil {
		q = q.Where("ticket_feedbacks.customer_id = ?", *filter.CustomerID)
	}
	if filter.CompanyID != nil {
		q = q.Where("ticket_feedbacks.company_id = ?", *filter.CompanyID)
	}
	if filter.Status != nil {
		q = q.Where("ticket_feedbacks.feedback_status = ?", *filter.Status)
	}
	if filter.Rating != nil {
		q = q.Where("ticket_feedbacks.rating = ?", *filter.Rating)
	}
	if filter.From != nil {
		q = q.Where("ticket_feedbacks.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("ticket_feedbacks.created_at <= ?", *filter.To)
	}
	if (filter.ServiceCallType != nil && *filter.ServiceCallType != "") ||
		(filter.Priority != nil && *filter.Priority != "") {
		q = q.Joins("JOIN tickets ON tickets.id = ticket_feedbacks.ticket_id")
		if filter.ServiceCallType != nil && *filter.ServiceCallType != "" {
			q = q.Where("tickets.service_call_type = ?", *filter.ServiceCallType)
		}
		if filter.Priority != nil && *filter.Priority != "" {
			q = q.Where("tickets.priority = ?", *filter.Priority)
		}
	}
	return q
}

func (r *FeedbackRepository) List(filter FeedbackListFilter) ([]models.TicketFeedback, int64, error) {
	q := r.db.Model(&models.TicketFeedback{})
	q = r.applyFilters(q, filter)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var rows []models.TicketFeedback
	err := r.applyFilters(r.db.Model(&models.TicketFeedback{}), filter).
		Preload("Engineer.User").
		Preload("Customer").
		Preload("Company").
		Preload("Ticket").
		Order("ticket_feedbacks.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	return rows, total, err
}

func (r *FeedbackRepository) ListPending(engineerID *uuid.UUID, limit int) ([]models.TicketFeedback, error) {
	q := r.db.Model(&models.TicketFeedback{}).
		Where("feedback_status = ?", models.FeedbackStatusPending)
	if engineerID != nil {
		q = q.Where("engineer_id = ?", *engineerID)
	}
	if limit <= 0 {
		limit = 100
	}
	var rows []models.TicketFeedback
	err := q.
		Preload("Engineer.User").
		Preload("Customer").
		Preload("Company").
		Preload("Ticket").
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// ListPendingByCustomer returns pending feedback rows owned by the given customer.
func (r *FeedbackRepository) ListPendingByCustomer(customerID uuid.UUID, limit int) ([]models.TicketFeedback, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.TicketFeedback
	err := r.db.Model(&models.TicketFeedback{}).
		Where("feedback_status = ? AND customer_id = ?", models.FeedbackStatusPending, customerID).
		Preload("Engineer.User").
		Preload("Ticket").
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *FeedbackRepository) ListByEngineer(engineerID uuid.UUID, limit int) ([]models.TicketFeedback, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []models.TicketFeedback
	err := r.db.
		Where("engineer_id = ? AND feedback_status = ?", engineerID, models.FeedbackStatusSubmitted).
		Preload("Customer").
		Preload("Company").
		Preload("Ticket").
		Order("submitted_at DESC NULLS LAST, created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

type EngineerFeedbackStats struct {
	EngineerID   uuid.UUID `json:"engineer_id"`
	Average      float64   `json:"average"`
	Total        int64     `json:"total"`
	FiveStar     int64     `json:"five_star"`
	FourStar     int64     `json:"four_star"`
	ThreeStar    int64     `json:"three_star"`
	TwoStar      int64     `json:"two_star"`
	OneStar      int64     `json:"one_star"`
	LowStar      int64     `json:"low_star"` // 1+2
	PendingCount int64     `json:"pending_count"`
}

func (r *FeedbackRepository) StatsForEngineer(engineerID uuid.UUID) (*EngineerFeedbackStats, error) {
	stats := &EngineerFeedbackStats{EngineerID: engineerID}

	type row struct {
		Average  float64
		Total    int64
		FiveStar int64
		FourStar int64
		ThreeStar int64
		TwoStar  int64
		OneStar  int64
	}
	var agg row
	err := r.db.Raw(`
		SELECT
			COALESCE(AVG(rating)::float, 0) AS average,
			COUNT(*) FILTER (WHERE feedback_status = 'Submitted' AND rating IS NOT NULL) AS total,
			COUNT(*) FILTER (WHERE rating = 5) AS five_star,
			COUNT(*) FILTER (WHERE rating = 4) AS four_star,
			COUNT(*) FILTER (WHERE rating = 3) AS three_star,
			COUNT(*) FILTER (WHERE rating = 2) AS two_star,
			COUNT(*) FILTER (WHERE rating = 1) AS one_star
		FROM ticket_feedbacks
		WHERE engineer_id = ? AND feedback_status = 'Submitted'
	`, engineerID).Scan(&agg).Error
	if err != nil {
		return nil, err
	}
	stats.Average = agg.Average
	stats.Total = agg.Total
	stats.FiveStar = agg.FiveStar
	stats.FourStar = agg.FourStar
	stats.ThreeStar = agg.ThreeStar
	stats.TwoStar = agg.TwoStar
	stats.OneStar = agg.OneStar
	stats.LowStar = agg.OneStar + agg.TwoStar

	if err := r.db.Model(&models.TicketFeedback{}).
		Where("engineer_id = ? AND feedback_status = ?", engineerID, models.FeedbackStatusPending).
		Count(&stats.PendingCount).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

type MonthlyRatingPoint struct {
	Month   string  `json:"month"`
	Average float64 `json:"average"`
	Count   int64   `json:"count"`
}

func (r *FeedbackRepository) MonthlyTrend(filter FeedbackListFilter) ([]MonthlyRatingPoint, error) {
	q := r.db.Table("ticket_feedbacks").
		Select(`TO_CHAR(DATE_TRUNC('month', COALESCE(submitted_at, created_at)), 'YYYY-MM') AS month,
			COALESCE(AVG(rating)::float, 0) AS average,
			COUNT(*) AS count`).
		Where("feedback_status = ? AND rating IS NOT NULL", models.FeedbackStatusSubmitted)
	q = r.applyFilters(q, filter)
	q = q.Group("month").Order("month ASC")

	var rows []MonthlyRatingPoint
	err := q.Scan(&rows).Error
	return rows, err
}

type RatingDistribution struct {
	Rating int   `json:"rating"`
	Count  int64 `json:"count"`
}

func (r *FeedbackRepository) Distribution(filter FeedbackListFilter) ([]RatingDistribution, error) {
	q := r.db.Table("ticket_feedbacks").
		Select("rating, COUNT(*) AS count").
		Where("feedback_status = ? AND rating IS NOT NULL", models.FeedbackStatusSubmitted)
	q = r.applyFilters(q, filter)
	q = q.Group("rating").Order("rating DESC")

	var rows []RatingDistribution
	err := q.Scan(&rows).Error
	return rows, err
}

type EngineerRankRow struct {
	EngineerID   uuid.UUID `json:"engineer_id"`
	EngineerName string    `json:"engineer_name"`
	Average      float64   `json:"average"`
	Total        int64     `json:"total"`
}

func (r *FeedbackRepository) EngineerRankings(filter FeedbackListFilter, minReviews int64, limit int, ascending bool) ([]EngineerRankRow, error) {
	if minReviews <= 0 {
		minReviews = 3
	}
	if limit <= 0 {
		limit = 10
	}
	order := "average DESC, total DESC"
	if ascending {
		order = "average ASC, total DESC"
	}

	q := r.db.Table("ticket_feedbacks").
		Select(`ticket_feedbacks.engineer_id AS engineer_id,
			COALESCE(NULLIF(TRIM(users.name), ''), users.email, 'Unknown') AS engineer_name,
			AVG(ticket_feedbacks.rating)::float AS average,
			COUNT(*) AS total`).
		Joins("JOIN support_engineers ON support_engineers.id = ticket_feedbacks.engineer_id").
		Joins("JOIN users ON users.id = support_engineers.user_id").
		Where("ticket_feedbacks.feedback_status = ? AND ticket_feedbacks.rating IS NOT NULL", models.FeedbackStatusSubmitted)
	q = r.applyFilters(q, filter)
	q = q.Group("ticket_feedbacks.engineer_id, users.name, users.email").
		Having("COUNT(*) >= ?", minReviews).
		Order(order).
		Limit(limit)

	var rows []EngineerRankRow
	err := q.Scan(&rows).Error
	return rows, err
}

type AnalyticsKPIs struct {
	AverageRating   float64 `json:"average_rating"`
	TotalSubmitted  int64   `json:"total_submitted"`
	PendingCount    int64   `json:"pending_count"`
	ResponseRate    float64 `json:"response_rate"`
	CSATPercent     float64 `json:"csat_percent"` // % of ratings >= 4
	FiveStarPercent float64 `json:"five_star_percent"`
}

func (r *FeedbackRepository) AnalyticsKPIs(filter FeedbackListFilter) (*AnalyticsKPIs, error) {
	kpis := &AnalyticsKPIs{}

	submittedQ := r.applyFilters(
		r.db.Table("ticket_feedbacks").Where("feedback_status = ? AND rating IS NOT NULL", models.FeedbackStatusSubmitted),
		filter,
	)
	type agg struct {
		Average  float64
		Total    int64
		Csat     int64
		FiveStar int64
	}
	var a agg
	if err := submittedQ.Select(`
		COALESCE(AVG(rating)::float, 0) AS average,
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE rating >= 4) AS csat,
		COUNT(*) FILTER (WHERE rating = 5) AS five_star
	`).Scan(&a).Error; err != nil {
		return nil, err
	}
	kpis.AverageRating = a.Average
	kpis.TotalSubmitted = a.Total
	if a.Total > 0 {
		kpis.CSATPercent = (float64(a.Csat) / float64(a.Total)) * 100
		kpis.FiveStarPercent = (float64(a.FiveStar) / float64(a.Total)) * 100
	}

	pendingQ := r.applyFilters(
		r.db.Table("ticket_feedbacks").Where("feedback_status = ?", models.FeedbackStatusPending),
		filter,
	)
	if err := pendingQ.Count(&kpis.PendingCount).Error; err != nil {
		return nil, err
	}

	totalRequests := kpis.TotalSubmitted + kpis.PendingCount
	if totalRequests > 0 {
		kpis.ResponseRate = (float64(kpis.TotalSubmitted) / float64(totalRequests)) * 100
	}
	return kpis, nil
}
