package service

import (
	"errors"

	"rbac/models"
	"rbac/repository"
	"rbac/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerService struct {
	db           *gorm.DB
	authRepo     *repository.AuthRepository
	customerRepo *repository.CustomerRepository
	ticketRepo   *repository.TicketRepository
	feedbackRepo *repository.FeedbackRepository
}

func NewCustomerService(
	db *gorm.DB,
	authRepo *repository.AuthRepository,
	customerRepo *repository.CustomerRepository,
	ticketRepo *repository.TicketRepository,
) *CustomerService {
	return &CustomerService{
		db:           db,
		authRepo:     authRepo,
		customerRepo: customerRepo,
		ticketRepo:   ticketRepo,
		feedbackRepo: repository.NewFeedbackRepository(db),
	}
}

func (s *CustomerService) attachFeedback(tickets []models.Ticket) []models.Ticket {
	if len(tickets) == 0 || s.feedbackRepo == nil {
		return tickets
	}
	ids := make([]string, 0, len(tickets))
	for _, t := range tickets {
		ids = append(ids, t.ID)
	}
	rows, err := s.feedbackRepo.GetByTicketIDs(ids)
	if err != nil {
		return tickets
	}
	for i := range tickets {
		if fb, ok := rows[tickets[i].ID]; ok {
			summary := models.TicketFeedbackSummary{
				ID:             fb.ID,
				FeedbackStatus: fb.FeedbackStatus,
				Rating:         fb.Rating,
				Remarks:        fb.Remarks,
				SubmittedAt:    fb.SubmittedAt,
				CreatedAt:      fb.CreatedAt,
			}
			tickets[i].Feedback = &summary
		}
	}
	return tickets
}

// =========================
// CREATE CUSTOMER (ADMIN FLOW)
// =========================
func (s *CustomerService) CreateCustomer(
	name string, // User name
	email string,
	password string,

	companyID uuid.UUID,
	companyName string, // Customer.Name

	contactPerson string, // ✅ ADD THIS
	location string,
	plant string,

	phone string,
	address string,
) error {

	return s.db.Transaction(func(tx *gorm.DB) error {

		// 1️⃣ Hash password
		hash, err := utils.HashPassword(password)
		if err != nil {
			return err
		}

		// 2️⃣ Create user
		user := &models.User{
			Name:     name,
			Email:    email,
			Password: string(hash),
			Role:     models.RoleCustomer,
			IsActive: true,
		}

		if err := s.authRepo.CreateUserTx(tx, user); err != nil {
			return err
		}

		// 3️⃣ Create customer profile
		customer := &models.Customer{
			UserID:        user.ID,
			CompanyID:     companyID,
			Name:          name, // Use the name parameter for customer name ✅ FIXED
			Address:       address,
			Location:      location,
			Plant:         plant,
			Phone:         phone,
			Email:         email,
			ContactPerson: contactPerson,
			IsActive:      true,
		}
		return s.customerRepo.Create(tx, customer)

	})
}

// =========================
// ADMIN: GET ALL CUSTOMERS
// =========================
func (s *CustomerService) GetAllCustomers(
	page int,
) ([]models.Customer, int64, error) {

	const limit = 3

	if page <= 0 {
		page = 1
	}

	return s.customerRepo.GetAllPaginated(page, limit)
}

// =========================
// ADMIN: GET CUSTOMER TICKETS
// =========================
func (s *CustomerService) GetCustomerTickets(
	customerID uuid.UUID,
) ([]models.Ticket, error) {
	tickets, err := s.ticketRepo.GetByCustomerID(customerID)
	if err != nil {
		return nil, err
	}
	return s.attachFeedback(tickets), nil
}

// =========================
// CUSTOMER: MY TICKETS
// =========================
func (s *CustomerService) GetMyTickets(
	userID uuid.UUID,
) ([]models.Ticket, error) {

	customer, err := s.customerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("customer profile not found")
	}

	tickets, err := s.ticketRepo.GetByCustomerID(customer.ID)
	if err != nil {
		return nil, err
	}
	return s.attachFeedback(tickets), nil
}

// =========================
// CUSTOMER: GET BY USER ID
// =========================
func (s *CustomerService) GetCustomerByUserID(
	userID uuid.UUID,
) (*models.Customer, error) {
	return s.customerRepo.GetByUserID(userID)
}
