package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
	"rbac/repository"
)

const maxFeedbackRemarksLen = 500

type FeedbackService struct {
	repo                *repository.FeedbackRepository
	ticketRepo          *repository.TicketRepository
	customerRepo        *repository.CustomerRepository
	supportEngineerRepo *repository.SupportEngineerRepository
	notificationService *NotificationService
	db                  *gorm.DB
}

func NewFeedbackService(
	r *repository.FeedbackRepository,
	ticketRepo *repository.TicketRepository,
	customerRepo *repository.CustomerRepository,
	supportEngineerRepo *repository.SupportEngineerRepository,
	notificationService *NotificationService,
	db *gorm.DB,
) *FeedbackService {
	return &FeedbackService{
		repo:                r,
		ticketRepo:          ticketRepo,
		customerRepo:        customerRepo,
		supportEngineerRepo: supportEngineerRepo,
		notificationService: notificationService,
		db:                  db,
	}
}

func sanitizeRemarks(raw string) string {
	s := strings.TrimSpace(raw)
	// Strip HTML tags to prevent stored XSS; React escapes on render.
	for {
		start := strings.Index(s, "<")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], ">")
		if end < 0 {
			s = s[:start]
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxFeedbackRemarksLen {
		runes := []rune(s)
		s = string(runes[:maxFeedbackRemarksLen])
	}
	return s
}

// EnsurePendingOnClose creates or refreshes a Pending feedback row when a ticket is closed.
func (s *FeedbackService) EnsurePendingOnClose(ticket *models.Ticket, engineerID uuid.UUID) error {
	if ticket == nil {
		return errors.New("ticket required")
	}
	if engineerID == uuid.Nil {
		return errors.New("engineer required for feedback")
	}

	customer, err := s.customerRepo.GetByID(ticket.CustomerID)
	if err != nil {
		// Fallback without is_active filter for closed-ticket feedback
		var c models.Customer
		if dbErr := s.db.Where("id = ?", ticket.CustomerID).First(&c).Error; dbErr != nil {
			return err
		}
		customer = &c
	}

	existing, err := s.repo.GetByTicketID(ticket.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := time.Now()
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		return s.repo.Create(&models.TicketFeedback{
			TicketID:       ticket.ID,
			EngineerID:     engineerID,
			CustomerID:     ticket.CustomerID,
			CompanyID:      customer.CompanyID,
			FeedbackStatus: models.FeedbackStatusPending,
			Remarks:        "",
			Metadata:       "{}",
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	if existing.FeedbackStatus == models.FeedbackStatusSubmitted {
		return nil
	}

	return s.repo.UpdateFields(existing.ID, map[string]interface{}{
		"engineer_id": engineerID,
		"customer_id": ticket.CustomerID,
		"company_id":  customer.CompanyID,
		"updated_at":  now,
	})
}

func (s *FeedbackService) Submit(
	userID uuid.UUID,
	ticketID string,
	rating int,
	remarks string,
) (*models.TicketFeedback, error) {
	if rating < 1 || rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	customer, err := s.customerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("ticket not found or access denied")
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, errors.New("ticket not found or access denied")
	}

	if ticket.CustomerID != customer.ID {
		return nil, errors.New("ticket not found or access denied")
	}

	if ticket.Status != models.StatusClosed {
		return nil, errors.New("feedback is only allowed on closed tickets")
	}

	fb, err := s.repo.GetByTicketID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if ticket.EngineerID == nil {
				return nil, errors.New("ticket has no assigned engineer")
			}
			now := time.Now()
			fb = &models.TicketFeedback{
				TicketID:       ticketID,
				EngineerID:     *ticket.EngineerID,
				CustomerID:     customer.ID,
				CompanyID:      customer.CompanyID,
				FeedbackStatus: models.FeedbackStatusPending,
				CreatedAt:      now,
				UpdatedAt:      now,
				Metadata:       "{}",
			}
			if createErr := s.repo.Create(fb); createErr != nil {
				return nil, errors.New("unable to submit feedback")
			}
			fb, err = s.repo.GetByTicketID(ticketID)
			if err != nil {
				return nil, errors.New("unable to submit feedback")
			}
		} else {
			return nil, errors.New("unable to submit feedback")
		}
	}

	if fb.FeedbackStatus == models.FeedbackStatusSubmitted {
		return nil, errors.New("feedback already submitted")
	}

	if fb.CustomerID != customer.ID {
		return nil, errors.New("ticket not found or access denied")
	}

	now := time.Now()
	safeRemarks := sanitizeRemarks(remarks)
	ratingCopy := rating

	if err := s.repo.UpdateFields(fb.ID, map[string]interface{}{
		"rating":          ratingCopy,
		"remarks":         safeRemarks,
		"feedback_status": models.FeedbackStatusSubmitted,
		"submitted_at":    now,
		"updated_at":      now,
		"engineer_id":     fb.EngineerID,
	}); err != nil {
		return nil, errors.New("unable to submit feedback")
	}

	updated, err := s.repo.GetByID(fb.ID)
	if err != nil {
		return nil, errors.New("unable to submit feedback")
	}

	if s.notificationService != nil {
		go s.notificationService.NotifyFeedbackReceived(updated)
	}

	return updated, nil
}

func (s *FeedbackService) GetByTicketID(userID uuid.UUID, role models.Role, ticketID string) (*models.TicketFeedback, error) {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, errors.New("ticket not found or access denied")
	}

	if err := s.authorizeTicketAccess(userID, role, ticket); err != nil {
		return nil, err
	}

	fb, err := s.repo.GetByTicketID(ticketID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("feedback not found")
		}
		return nil, err
	}
	return fb, nil
}

func (s *FeedbackService) authorizeTicketAccess(userID uuid.UUID, role models.Role, ticket *models.Ticket) error {
	switch role {
	case models.RoleAdmin:
		return nil
	case models.RoleCustomer:
		customer, err := s.customerRepo.GetByUserID(userID)
		if err != nil || customer.ID != ticket.CustomerID {
			return errors.New("ticket not found or access denied")
		}
		return nil
	case models.RoleSupport:
		eng, err := s.supportEngineerRepo.GetByUserID(userID)
		if err != nil {
			return errors.New("ticket not found or access denied")
		}
		if ticket.EngineerID == nil || *ticket.EngineerID != eng.ID {
			return errors.New("ticket not found or access denied")
		}
		return nil
	default:
		return errors.New("ticket not found or access denied")
	}
}

func (s *FeedbackService) GetMyEngineerFeedback(requesterID uuid.UUID) (map[string]interface{}, error) {
	eng, err := s.supportEngineerRepo.GetByUserID(requesterID)
	if err != nil {
		return nil, errors.New("access denied")
	}
	return s.GetEngineerFeedback(requesterID, models.RoleSupport, eng.ID)
}

func (s *FeedbackService) GetEngineerFeedback(requesterID uuid.UUID, role models.Role, engineerID uuid.UUID) (map[string]interface{}, error) {
	if role == models.RoleSupport {
		eng, err := s.supportEngineerRepo.GetByUserID(requesterID)
		if err != nil || eng.ID != engineerID {
			return nil, errors.New("access denied")
		}
	} else if role != models.RoleAdmin {
		return nil, errors.New("access denied")
	}

	stats, err := s.repo.StatsForEngineer(engineerID)
	if err != nil {
		return nil, err
	}
	recent, err := s.repo.ListByEngineer(engineerID, 20)
	if err != nil {
		return nil, err
	}
	trend, err := s.repo.MonthlyTrend(repository.FeedbackListFilter{EngineerID: &engineerID})
	if err != nil {
		return nil, err
	}
	dist, err := s.repo.Distribution(repository.FeedbackListFilter{EngineerID: &engineerID})
	if err != nil {
		return nil, err
	}

	fiveStarPct := 0.0
	if stats.Total > 0 {
		fiveStarPct = (float64(stats.FiveStar) / float64(stats.Total)) * 100
	}

	return map[string]interface{}{
		"stats": map[string]interface{}{
			"average_rating":    stats.Average,
			"total_feedback":    stats.Total,
			"five_star_percent": fiveStarPct,
			"low_star_count":    stats.LowStar,
			"pending_count":     stats.PendingCount,
			"distribution": map[string]int64{
				"5": stats.FiveStar,
				"4": stats.FourStar,
				"3": stats.ThreeStar,
				"2": stats.TwoStar,
				"1": stats.OneStar,
			},
		},
		"monthly_trend": trend,
		"distribution":  dist,
		"recent":        recent,
	}, nil
}

func (s *FeedbackService) ListPending(requesterID uuid.UUID, role models.Role) ([]models.TicketFeedback, error) {
	switch role {
	case models.RoleAdmin:
		return s.repo.ListPending(nil, 200)
	case models.RoleSupport:
		eng, err := s.supportEngineerRepo.GetByUserID(requesterID)
		if err != nil {
			return nil, errors.New("access denied")
		}
		return s.repo.ListPending(&eng.ID, 100)
	default:
		return nil, errors.New("access denied")
	}
}

func (s *FeedbackService) Analytics(filter repository.FeedbackListFilter) (map[string]interface{}, error) {
	kpis, err := s.repo.AnalyticsKPIs(filter)
	if err != nil {
		return nil, err
	}
	trend, err := s.repo.MonthlyTrend(filter)
	if err != nil {
		return nil, err
	}
	dist, err := s.repo.Distribution(filter)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.EngineerRankings(filter, 3, 10, false)
	if err != nil {
		return nil, err
	}
	lowest, err := s.repo.EngineerRankings(filter, 3, 10, true)
	if err != nil {
		return nil, err
	}
	ranked, err := s.repo.EngineerRankings(filter, 1, 50, false)
	if err != nil {
		return nil, err
	}
	recent, _, err := s.repo.List(repository.FeedbackListFilter{
		EngineerID:      filter.EngineerID,
		CustomerID:      filter.CustomerID,
		CompanyID:       filter.CompanyID,
		Status:          statusPtrSubmitted(),
		Rating:          filter.Rating,
		ServiceCallType: filter.ServiceCallType,
		Priority:        filter.Priority,
		From:            filter.From,
		To:              filter.To,
		Limit:           50,
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"kpis":             kpis,
		"monthly_trend":    trend,
		"distribution":     dist,
		"top_engineers":    top,
		"lowest_engineers": lowest,
		"ranked_engineers": ranked,
		"recent":           recent,
	}, nil
}

func statusPtrSubmitted() *models.FeedbackStatus {
	s := models.FeedbackStatusSubmitted
	return &s
}

func (s *FeedbackService) Reopen(feedbackID uuid.UUID) (*models.TicketFeedback, error) {
	fb, err := s.repo.GetByID(feedbackID)
	if err != nil {
		return nil, errors.New("feedback not found")
	}
	if fb.FeedbackStatus != models.FeedbackStatusSubmitted {
		return nil, errors.New("only submitted feedback can be reopened")
	}
	now := time.Now()
	if err := s.repo.UpdateFields(fb.ID, map[string]interface{}{
		"feedback_status": models.FeedbackStatusPending,
		"rating":          nil,
		"remarks":         "",
		"submitted_at":    nil,
		"updated_at":      now,
	}); err != nil {
		return nil, err
	}
	return s.repo.GetByID(fb.ID)
}

func (s *FeedbackService) SummariesForTickets(ticketIDs []string) (map[string]models.TicketFeedbackSummary, error) {
	rows, err := s.repo.GetByTicketIDs(ticketIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]models.TicketFeedbackSummary, len(rows))
	for id, fb := range rows {
		out[id] = models.TicketFeedbackSummary{
			ID:             fb.ID,
			FeedbackStatus: fb.FeedbackStatus,
			Rating:         fb.Rating,
			Remarks:        fb.Remarks,
			SubmittedAt:    fb.SubmittedAt,
			CreatedAt:      fb.CreatedAt,
		}
	}
	return out, nil
}

func AttachFeedbackSummary(ticket *models.Ticket, summary *models.TicketFeedbackSummary) {
	if ticket == nil || summary == nil {
		return
	}
	ticket.Feedback = summary
}
