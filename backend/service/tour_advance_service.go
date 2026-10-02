package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"rbac/domain"
	"rbac/models"
)

var (
	ErrTourNotFound      = errors.New("request not found")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrTourForbidden     = errors.New("insufficient permissions")
)

type TourAdvanceService struct {
	db            *gorm.DB
	notifications *NotificationService
}

func NewTourAdvanceService(db *gorm.DB, notifications *NotificationService) *TourAdvanceService {
	return &TourAdvanceService{db: db, notifications: notifications}
}

type CreateTourAdvanceInput struct {
	CustomerID           uuid.UUID
	WorkType             string
	PONumber             string
	TicketID             string
	PersonsTravelling    int
	DaysPlanned          int
	FoodExpense          string
	LocalExpense         string
	AccommodationExpense string
	TravelMode           string
	TravelExpense        string
}

type TourAdvanceEventView struct {
	Action    string    `json:"action"`
	ActorName string    `json:"actor_name"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type TourAdvanceView struct {
	ID                   uuid.UUID              `json:"id"`
	RequestNumber        string                 `json:"request_number"`
	EmployeeName         string                 `json:"employee_name"`
	CustomerID           uuid.UUID              `json:"customer_id"`
	CustomerName         string                 `json:"customer_name"`
	Location             string                 `json:"location"`
	WorkType             string                 `json:"work_type"`
	PONumber             *string                `json:"po_number"`
	TicketID             *string                `json:"ticket_id"`
	PersonsTravelling    int                    `json:"persons_travelling"`
	DaysPlanned          int                    `json:"days_planned"`
	FoodExpense          string                 `json:"food_expense"`
	LocalExpense         string                 `json:"local_expense"`
	AccommodationExpense string                 `json:"accommodation_expense"`
	TravelMode           string                 `json:"travel_mode"`
	TravelExpense        string                 `json:"travel_expense"`
	RequestedAmount      string                 `json:"requested_amount"`
	ApprovedAmount       *string                `json:"approved_amount"`
	Status               string                 `json:"status"`
	ApprovalRemarks      string                 `json:"approval_remarks"`
	ApprovedByName       string                 `json:"approved_by_name"`
	ApprovedAt           *time.Time             `json:"approved_at"`
	RejectionRemarks     string                 `json:"rejection_remarks"`
	RejectedByName       string                 `json:"rejected_by_name"`
	RejectedAt           *time.Time             `json:"rejected_at"`
	BillSubmittedByName  string                 `json:"bill_submitted_by_name"`
	BillSubmittedAt      *time.Time             `json:"bill_submitted_at"`
	ProcessedByName      string                 `json:"processed_by_name"`
	ProcessedAt          *time.Time             `json:"processed_at"`
	CreatedAt            time.Time              `json:"created_at"`
	Events               []TourAdvanceEventView `json:"events"`
}

type TourAdvanceList struct {
	Requests []TourAdvanceView `json:"requests"`
	Counts   map[string]int64  `json:"counts,omitempty"`
}

type CompanyOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type SiteOption struct {
	ID       uuid.UUID `json:"id"`
	Location string    `json:"location"`
	Plant    string    `json:"plant"`
	Name     string    `json:"name"`
	Label    string    `json:"label"`
}

type TicketOption struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (s *TourAdvanceService) Create(actor uuid.UUID, role models.Role, in CreateTourAdvanceInput) (*TourAdvanceView, error) {
	if role != models.RoleSupport {
		return nil, ErrTourForbidden
	}

	var engineer models.SupportEngineer
	if err := s.db.Where("user_id = ? AND is_active = ?", actor, true).First(&engineer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTourForbidden
		}
		return nil, err
	}

	var customer models.Customer
	err := s.db.Preload("Company").Where("id = ? AND is_active = ?", in.CustomerID, true).First(&customer).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &domain.RuleError{Msg: "customer not found"}
	}
	if err != nil {
		return nil, err
	}

	location := strings.TrimSpace(customer.Location)
	if location == "" {
		location = strings.TrimSpace(customer.Plant)
	}

	ticketID := strings.TrimSpace(in.TicketID)
	hasTicket := ticketID != ""
	if hasTicket {
		var ticket models.Ticket
		err := s.db.Select("id", "customer_id").Where("id = ?", ticketID).First(&ticket).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &domain.RuleError{Msg: "ticket not found"}
		}
		if err != nil {
			return nil, err
		}
		if ticket.CustomerID != customer.ID {
			return nil, &domain.RuleError{Msg: "ticket does not belong to the selected customer"}
		}
	}

	food, err := domain.ParseMoney(in.FoodExpense)
	if err != nil {
		return nil, err
	}
	local, err := domain.ParseMoney(in.LocalExpense)
	if err != nil {
		return nil, err
	}
	accommodation, err := domain.ParseMoney(in.AccommodationExpense)
	if err != nil {
		return nil, err
	}
	travel, err := domain.ParseMoney(in.TravelExpense)
	if err != nil {
		return nil, err
	}
	expenses := domain.TourExpenseInput{
		Food: food, Local: local, Accommodation: accommodation, Travel: travel,
	}
	rules := domain.TourCreateRules{
		WorkType:   models.TourWorkType(strings.ToUpper(strings.TrimSpace(in.WorkType))),
		PONumber:   in.PONumber,
		HasTicket:  hasTicket,
		Location:   location,
		Persons:    in.PersonsTravelling,
		Days:       in.DaysPlanned,
		TravelMode: models.TourTravelMode(strings.ToUpper(strings.TrimSpace(in.TravelMode))),
		Expenses:   expenses,
	}
	if err := domain.ValidateTourCreate(rules); err != nil {
		return nil, err
	}
	requested, err := domain.RequestedAmount(expenses)
	if err != nil {
		return nil, err
	}

	var poPtr *string
	var ticketPtr *string
	if rules.WorkType == models.TourWorkInstallation {
		po := strings.TrimSpace(in.PONumber)
		poPtr = &po
	} else {
		ticketPtr = &ticketID
	}

	row := &models.TourAdvanceRequest{
		ID:                   uuid.New(),
		EngineerID:           engineer.ID,
		CustomerID:           customer.ID,
		LocationSnapshot:     location,
		WorkType:             rules.WorkType,
		PONumber:             poPtr,
		TicketID:             ticketPtr,
		PersonsTravelling:    in.PersonsTravelling,
		DaysPlanned:          in.DaysPlanned,
		FoodExpense:          food,
		LocalExpense:         local,
		AccommodationExpense: accommodation,
		TravelMode:           rules.TravelMode,
		TravelExpense:        travel,
		RequestedAmount:      requested,
		Status:               models.TourStatusPendingApproval,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		number, err := nextTourRequestNumber(tx)
		if err != nil {
			return err
		}
		row.RequestNumber = number
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return tx.Create(&models.TourAdvanceEvent{
			ID:                   uuid.New(),
			TourAdvanceRequestID: row.ID,
			Action:               models.TourEventCreated,
			ActorUserID:          actor,
			CreatedAt:            time.Now(),
		}).Error
	})
	if err != nil {
		return nil, err
	}

	var actorUser models.User
	_ = s.db.Select("name").First(&actorUser, "id = ?", actor).Error
	s.notifyRole(
		models.RoleSuperAdmin,
		models.NotificationTypeTourAdvanceSubmitted,
		"Tour advance submitted",
		fmt.Sprintf("%s submitted %s for approval.", actorUser.Name, row.RequestNumber),
		row,
	)

	return s.Get(actor, role, row.ID)
}

func (s *TourAdvanceService) List(actor uuid.UUID, role models.Role, status string) (*TourAdvanceList, error) {
	status = strings.TrimSpace(status)
	q := s.db.Model(&models.TourAdvanceRequest{})
	switch role {
	case models.RoleSupport:
		var engineer models.SupportEngineer
		if err := s.db.Where("user_id = ?", actor).First(&engineer).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &TourAdvanceList{Requests: []TourAdvanceView{}}, nil
			}
			return nil, err
		}
		q = q.Where("engineer_id = ?", engineer.ID)
	case models.RoleAdmin:
		q = q.Where("status IN ?", []models.TourAdvanceStatus{
			models.TourStatusBillSubmitted,
			models.TourStatusProcessed,
		})
	case models.RoleSuperAdmin:
	default:
		return nil, ErrTourForbidden
	}

	counts, err := s.countByStatus(q.Session(&gorm.Session{}))
	if err != nil {
		return nil, err
	}

	filtered := q.Session(&gorm.Session{})
	if status != "" {
		if !knownTourStatus(status) {
			return nil, &domain.RuleError{Msg: "invalid status"}
		}
		if role == models.RoleAdmin && status != string(models.TourStatusBillSubmitted) && status != string(models.TourStatusProcessed) {
			return &TourAdvanceList{Requests: []TourAdvanceView{}, Counts: counts}, nil
		}
		filtered = filtered.Where("status = ?", status)
	}

	var rows []models.TourAdvanceRequest
	if err := s.withViewPreloads(filtered).Order("created_at DESC").Limit(500).Find(&rows).Error; err != nil {
		return nil, err
	}
	views := make([]TourAdvanceView, 0, len(rows))
	for i := range rows {
		views = append(views, toTourView(&rows[i]))
	}
	out := &TourAdvanceList{Requests: views}
	if role != models.RoleSupport {
		out.Counts = counts
	}
	return out, nil
}

func (s *TourAdvanceService) Get(actor uuid.UUID, role models.Role, id uuid.UUID) (*TourAdvanceView, error) {
	row, err := s.load(id)
	if err != nil {
		return nil, err
	}
	if !s.canView(actor, role, row) {
		return nil, ErrTourNotFound
	}
	view := toTourView(row)
	return &view, nil
}

func (s *TourAdvanceService) Approve(actor uuid.UUID, role models.Role, id uuid.UUID, amountRaw, remarks string) (*TourAdvanceView, error) {
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

	err = s.transition(id, models.TourStatusPendingApproval, models.TourStatusApproved, func(tx *gorm.DB, row *models.TourAdvanceRequest) error {
		if err := domain.ValidateApprovedAmount(row.RequestedAmount, amount); err != nil {
			return err
		}
		now := time.Now()
		res := tx.Model(&models.TourAdvanceRequest{}).
			Where("id = ? AND status = ?", id, models.TourStatusPendingApproval).
			Updates(map[string]interface{}{
				"status":           models.TourStatusApproved,
				"approved_amount":  amount,
				"approval_remarks": remarks,
				"approved_by":      actor,
				"approved_at":      now,
				"updated_at":       now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidTransition
		}
		return writeTourEvent(tx, id, models.TourEventApproved, actor, remarks)
	})
	if err != nil {
		return nil, err
	}
	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyEngineer(id, models.NotificationTypeTourAdvanceApproved, "Tour advance approved",
		fmt.Sprintf("%s was approved.", view.RequestNumber))
	return view, nil
}

func (s *TourAdvanceService) Reject(actor uuid.UUID, role models.Role, id uuid.UUID, remarks string) (*TourAdvanceView, error) {
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

	err := s.transition(id, models.TourStatusPendingApproval, models.TourStatusRejected, func(tx *gorm.DB, row *models.TourAdvanceRequest) error {
		now := time.Now()
		res := tx.Model(&models.TourAdvanceRequest{}).
			Where("id = ? AND status = ?", id, models.TourStatusPendingApproval).
			Updates(map[string]interface{}{
				"status":            models.TourStatusRejected,
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
		return writeTourEvent(tx, id, models.TourEventRejected, actor, remarks)
	})
	if err != nil {
		return nil, err
	}
	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyEngineer(id, models.NotificationTypeTourAdvanceRejected, "Tour advance rejected",
		fmt.Sprintf("%s was rejected.", view.RequestNumber))
	return view, nil
}

func (s *TourAdvanceService) Delete(actor uuid.UUID, role models.Role, id uuid.UUID) error {
	if role != models.RoleSupport {
		return ErrTourForbidden
	}
	row, err := s.load(id)
	if err != nil {
		return err
	}
	if row.Engineer == nil || row.Engineer.UserID != actor {
		return ErrTourNotFound
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(
			"type IN ? AND metadata LIKE ?",
			[]models.NotificationType{
				models.NotificationTypeTourAdvanceSubmitted,
				models.NotificationTypeTourAdvanceApproved,
				models.NotificationTypeTourAdvanceRejected,
				models.NotificationTypeTourAdvanceBillSubmitted,
				models.NotificationTypeTourAdvanceProcessed,
			},
			"%"+id.String()+"%",
		).Delete(&models.Notification{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tour_advance_request_id = ?", id).Delete(&models.TourAdvanceEvent{}).Error; err != nil {
			return err
		}
		res := tx.Where("id = ?", id).Delete(&models.TourAdvanceRequest{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrTourNotFound
		}
		return nil
	})
}

func (s *TourAdvanceService) MarkBillSubmitted(actor uuid.UUID, role models.Role, id uuid.UUID) (*TourAdvanceView, error) {
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

	err = s.transition(id, models.TourStatusApproved, models.TourStatusBillSubmitted, func(tx *gorm.DB, current *models.TourAdvanceRequest) error {
		now := time.Now()
		res := tx.Model(&models.TourAdvanceRequest{}).
			Where("id = ? AND status = ?", id, models.TourStatusApproved).
			Updates(map[string]interface{}{
				"status":            models.TourStatusBillSubmitted,
				"bill_submitted_by": actor,
				"bill_submitted_at": now,
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidTransition
		}
		return writeTourEvent(tx, id, models.TourEventBillSubmitted, actor, "")
	})
	if err != nil {
		return nil, err
	}
	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyRole(models.RoleAdmin, models.NotificationTypeTourAdvanceBillSubmitted, "Bills submitted",
		fmt.Sprintf("%s is ready to process.", view.RequestNumber), row)
	return view, nil
}

func (s *TourAdvanceService) Process(actor uuid.UUID, role models.Role, id uuid.UUID) (*TourAdvanceView, error) {
	if role != models.RoleAdmin {
		return nil, ErrTourForbidden
	}
	existing, err := s.load(id)
	if err != nil {
		return nil, err
	}
	if existing.Status != models.TourStatusBillSubmitted && existing.Status != models.TourStatusProcessed {
		return nil, ErrTourNotFound
	}
	err = s.transition(id, models.TourStatusBillSubmitted, models.TourStatusProcessed, func(tx *gorm.DB, current *models.TourAdvanceRequest) error {
		now := time.Now()
		res := tx.Model(&models.TourAdvanceRequest{}).
			Where("id = ? AND status = ?", id, models.TourStatusBillSubmitted).
			Updates(map[string]interface{}{
				"status":       models.TourStatusProcessed,
				"processed_by": actor,
				"processed_at": now,
				"updated_at":   now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidTransition
		}
		return writeTourEvent(tx, id, models.TourEventProcessed, actor, "")
	})
	if err != nil {
		return nil, err
	}
	view, err := s.Get(actor, role, id)
	if err != nil {
		return nil, err
	}
	s.notifyEngineer(id, models.NotificationTypeTourAdvanceProcessed, "Tour advance processed",
		fmt.Sprintf("%s has been processed.", view.RequestNumber))
	return view, nil
}

func (s *TourAdvanceService) ListCompanies() ([]CompanyOption, error) {
	var rows []models.Company
	if err := s.db.Where("is_active = ?", true).Order("name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]CompanyOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, CompanyOption{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func (s *TourAdvanceService) ListSites(companyID uuid.UUID) ([]SiteOption, error) {
	if companyID == uuid.Nil {
		return nil, &domain.RuleError{Msg: "customer is required"}
	}
	var company models.Company
	if err := s.db.Where("id = ? AND is_active = ?", companyID, true).First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &domain.RuleError{Msg: "customer not found"}
		}
		return nil, err
	}
	var rows []models.Customer
	if err := s.db.Where("company_id = ? AND is_active = ?", companyID, true).Order("location ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SiteOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, SiteOption{
			ID:       row.ID,
			Location: row.Location,
			Plant:    row.Plant,
			Name:     row.Name,
			Label:    siteLabel(row),
		})
	}
	return out, nil
}

func (s *TourAdvanceService) ListTickets(customerID uuid.UUID) ([]TicketOption, error) {
	if customerID == uuid.Nil {
		return nil, &domain.RuleError{Msg: "customer is required"}
	}
	var customer models.Customer
	if err := s.db.Where("id = ? AND is_active = ?", customerID, true).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &domain.RuleError{Msg: "customer not found"}
		}
		return nil, err
	}
	var rows []models.Ticket
	if err := s.db.Select("id", "title").Where("customer_id = ?", customerID).Order("created_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TicketOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, TicketOption{ID: row.ID, Title: row.Title})
	}
	return out, nil
}

func (s *TourAdvanceService) transition(id uuid.UUID, from, to models.TourAdvanceStatus, apply func(tx *gorm.DB, row *models.TourAdvanceRequest) error) error {
	if !domain.CanTourTransition(from, to) {
		return ErrInvalidTransition
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		q := tx
		if tx.Dialector.Name() == "postgres" {
			q = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var row models.TourAdvanceRequest
		err := q.First(&row, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTourNotFound
		}
		if err != nil {
			return err
		}
		if !domain.CanTourTransition(row.Status, to) {
			return ErrInvalidTransition
		}
		return apply(tx, &row)
	})
}

func (s *TourAdvanceService) load(id uuid.UUID) (*models.TourAdvanceRequest, error) {
	var row models.TourAdvanceRequest
	err := s.withViewPreloads(s.db).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTourNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *TourAdvanceService) withViewPreloads(db *gorm.DB) *gorm.DB {
	return db.Preload("Engineer.User").
		Preload("Customer.Company").
		Preload("ApprovedByUser").
		Preload("RejectedByUser").
		Preload("BillSubmittedByUser").
		Preload("ProcessedByUser").
		Preload("Events", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("created_at ASC")
		}).
		Preload("Events.Actor")
}

func (s *TourAdvanceService) canView(actor uuid.UUID, role models.Role, row *models.TourAdvanceRequest) bool {
	switch role {
	case models.RoleSuperAdmin:
		return true
	case models.RoleAdmin:
		return row.Status == models.TourStatusBillSubmitted || row.Status == models.TourStatusProcessed
	case models.RoleSupport:
		return row.Engineer != nil && row.Engineer.UserID == actor
	default:
		return false
	}
}

func (s *TourAdvanceService) countByStatus(scope *gorm.DB) (map[string]int64, error) {
	type row struct {
		Status string
		Count  int64
	}
	var rows []row
	if err := scope.Session(&gorm.Session{}).Select("status, COUNT(*) AS count").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[string]int64{
		string(models.TourStatusPendingApproval): 0,
		string(models.TourStatusApproved):        0,
		string(models.TourStatusRejected):        0,
		string(models.TourStatusBillSubmitted):   0,
		string(models.TourStatusProcessed):       0,
	}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

func (s *TourAdvanceService) notifyRole(role models.Role, ntype models.NotificationType, title, message string, row *models.TourAdvanceRequest) {
	if s.notifications == nil || row == nil {
		return
	}
	var users []models.User
	if err := s.db.Where("role = ? AND is_active = ?", role, true).Find(&users).Error; err != nil {
		log.Printf("tour advance notify: %v", err)
		return
	}
	meta := tourMeta(row)
	for _, user := range users {
		if err := s.notifications.CreateInApp(user.ID, ntype, title, message, meta); err != nil {
			log.Printf("tour advance notify user %s: %v", user.ID, err)
		}
	}
}

func (s *TourAdvanceService) notifyEngineer(requestID uuid.UUID, ntype models.NotificationType, title, message string) {
	if s.notifications == nil {
		return
	}
	var row models.TourAdvanceRequest
	if err := s.db.Preload("Engineer").First(&row, "id = ?", requestID).Error; err != nil {
		log.Printf("tour advance notify engineer: %v", err)
		return
	}
	if row.Engineer == nil {
		return
	}
	if err := s.notifications.CreateInApp(row.Engineer.UserID, ntype, title, message, tourMeta(&row)); err != nil {
		log.Printf("tour advance notify engineer: %v", err)
	}
}

func nextTourRequestNumber(tx *gorm.DB) (string, error) {
	if tx.Dialector.Name() != "postgres" {
		var count int64
		if err := tx.Model(&models.TourAdvanceRequest{}).Count(&count).Error; err != nil {
			return "", err
		}
		return fmt.Sprintf("TR-%06d", count+1), nil
	}
	var n int64
	if err := tx.Raw("SELECT nextval('tour_advance_request_number_seq')").Scan(&n).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("TR-%06d", n), nil
}

func writeTourEvent(tx *gorm.DB, requestID uuid.UUID, action models.TourAdvanceEventAction, actor uuid.UUID, note string) error {
	return tx.Create(&models.TourAdvanceEvent{
		ID:                   uuid.New(),
		TourAdvanceRequestID: requestID,
		Action:               action,
		ActorUserID:          actor,
		Note:                 note,
		CreatedAt:            time.Now(),
	}).Error
}

func toTourView(row *models.TourAdvanceRequest) TourAdvanceView {
	view := TourAdvanceView{
		ID:                   row.ID,
		RequestNumber:        row.RequestNumber,
		EmployeeName:         employeeName(row),
		CustomerID:           row.CustomerID,
		CustomerName:         customerDisplayName(row),
		Location:             row.LocationSnapshot,
		WorkType:             string(row.WorkType),
		PONumber:             row.PONumber,
		TicketID:             row.TicketID,
		PersonsTravelling:    row.PersonsTravelling,
		DaysPlanned:          row.DaysPlanned,
		FoodExpense:          row.FoodExpense.StringFixed(2),
		LocalExpense:         row.LocalExpense.StringFixed(2),
		AccommodationExpense: row.AccommodationExpense.StringFixed(2),
		TravelMode:           string(row.TravelMode),
		TravelExpense:        row.TravelExpense.StringFixed(2),
		RequestedAmount:      row.RequestedAmount.StringFixed(2),
		Status:               string(row.Status),
		ApprovalRemarks:      row.ApprovalRemarks,
		ApprovedByName:       userName(row.ApprovedByUser),
		ApprovedAt:           row.ApprovedAt,
		RejectionRemarks:     row.RejectionRemarks,
		RejectedByName:       userName(row.RejectedByUser),
		RejectedAt:           row.RejectedAt,
		BillSubmittedByName:  userName(row.BillSubmittedByUser),
		BillSubmittedAt:      row.BillSubmittedAt,
		ProcessedByName:      userName(row.ProcessedByUser),
		ProcessedAt:          row.ProcessedAt,
		CreatedAt:            row.CreatedAt,
		Events:               make([]TourAdvanceEventView, 0, len(row.Events)),
	}
	if row.ApprovedAmount != nil {
		formatted := row.ApprovedAmount.StringFixed(2)
		view.ApprovedAmount = &formatted
	}
	for _, event := range row.Events {
		view.Events = append(view.Events, TourAdvanceEventView{
			Action:    string(event.Action),
			ActorName: userName(event.Actor),
			Note:      event.Note,
			CreatedAt: event.CreatedAt,
		})
	}
	return view
}

func employeeName(row *models.TourAdvanceRequest) string {
	if row == nil || row.Engineer == nil {
		return ""
	}
	return row.Engineer.User.Name
}

func customerDisplayName(row *models.TourAdvanceRequest) string {
	if row == nil || row.Customer == nil {
		return ""
	}
	if name := strings.TrimSpace(row.Customer.Company.Name); name != "" {
		return name
	}
	return row.Customer.Name
}

func userName(user *models.User) string {
	if user == nil {
		return ""
	}
	return user.Name
}

func siteLabel(row models.Customer) string {
	location := strings.TrimSpace(row.Location)
	plant := strings.TrimSpace(row.Plant)
	switch {
	case location != "" && plant != "" && location != plant:
		return location + " · " + plant
	case location != "":
		return location
	case plant != "":
		return plant
	default:
		return strings.TrimSpace(row.Name)
	}
}

func knownTourStatus(status string) bool {
	switch models.TourAdvanceStatus(status) {
	case models.TourStatusPendingApproval, models.TourStatusApproved, models.TourStatusRejected, models.TourStatusBillSubmitted, models.TourStatusProcessed:
		return true
	default:
		return false
	}
}

func tourMeta(row *models.TourAdvanceRequest) string {
	if row == nil {
		return "{}"
	}
	raw, err := json.Marshal(map[string]string{
		"tour_advance_id": row.ID.String(),
		"request_number":  row.RequestNumber,
	})
	if err != nil {
		return "{}"
	}
	return string(raw)
}
