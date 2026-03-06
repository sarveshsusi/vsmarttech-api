// service/admin_service.go
package service

import (
	"time"

	"rbac/models"
	"rbac/repository"
)

type AdminService struct {
	repo *repository.DashboardRepository
}

func NewAdminService(r *repository.DashboardRepository) *AdminService {
	return &AdminService{repo: r}
}

func (s *AdminService) GetDashboardStats() (map[string]int64, error) {
	return s.repo.FetchAdminStats()
}

func (s *AdminService) GetDashboardTickets(
	companyID *string,
	contractType *string,
	status *string,
	startDate *time.Time,
	endDate *time.Time,
	limit int,
	offset int,
) ([]models.Ticket, int64, error) {
	return s.repo.GetDashboardTickets(companyID, contractType, status, startDate, endDate, limit, offset)
}

// ========================
// SUPPORT DASHBOARD
// ========================

func (s *AdminService) GetSupportDashboardStats() (*repository.SupportStats, error) {
	return s.repo.GetSupportStats()
}

// ========================
// CUSTOMER DASHBOARD
// ========================

func (s *AdminService) GetCustomerDashboardStats(customerID string) (*repository.CustomerStats, error) {
	return s.repo.GetCustomerStats(customerID)
}

func (s *AdminService) GetCustomerTicketCheckpoints(customerID string) ([]repository.TicketCheckpoint, error) {
	return s.repo.GetCustomerTickets(customerID)
}
