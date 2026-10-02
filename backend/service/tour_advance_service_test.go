package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"rbac/domain"
	"rbac/models"
)

type tourFixture struct {
	db            *gorm.DB
	engineer      models.User
	otherEngineer models.User
	admin         models.User
	superAdmin    models.User
	siteA         models.Customer
	siteB         models.Customer
	ticketA       models.Ticket
	ticketB       models.Ticket
}

func openTourDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.Exec(tourTestSchema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func seedTour(t *testing.T, db *gorm.DB) tourFixture {
	t.Helper()
	fx := tourFixture{db: db}
	company := models.Company{ID: uuid.New(), Name: "TVARUR Oils And Fats Private Limited", IsActive: true}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	fx.engineer = mustUser(t, db, "Sarvesh", models.RoleSupport)
	fx.otherEngineer = mustUser(t, db, "Other", models.RoleSupport)
	fx.admin = mustUser(t, db, "Admin", models.RoleAdmin)
	fx.superAdmin = mustUser(t, db, "Super Admin", models.RoleSuperAdmin)
	mustEngineer(t, db, fx.engineer.ID)
	mustEngineer(t, db, fx.otherEngineer.ID)

	fx.siteA = models.Customer{
		ID: uuid.New(), UserID: mustUser(t, db, "Site A", models.RoleCustomer).ID,
		CompanyID: company.ID, Name: "Site A", Location: "TVARUR", Plant: "Plant A", IsActive: true,
	}
	fx.siteB = models.Customer{
		ID: uuid.New(), UserID: mustUser(t, db, "Site B", models.RoleCustomer).ID,
		CompanyID: company.ID, Name: "Site B", Location: "OTHER", Plant: "Plant B", IsActive: true,
	}
	if err := db.Create(&fx.siteA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&fx.siteB).Error; err != nil {
		t.Fatal(err)
	}
	fx.ticketA = models.Ticket{ID: "VS/10/26/1", CustomerID: fx.siteA.ID, Title: "Pump", Status: models.StatusOpen, CreatedBy: fx.admin.ID, CreatedAt: time.Now()}
	fx.ticketB = models.Ticket{ID: "VS/10/26/2", CustomerID: fx.siteB.ID, Title: "Motor", Status: models.StatusOpen, CreatedBy: fx.admin.ID, CreatedAt: time.Now()}
	closed := models.Ticket{ID: "VS/10/26/3", CustomerID: fx.siteA.ID, Title: "Done", Status: models.StatusClosed, CreatedBy: fx.admin.ID, CreatedAt: time.Now()}
	if err := db.Create(&fx.ticketA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&fx.ticketB).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&closed).Error; err != nil {
		t.Fatal(err)
	}
	return fx
}

func mustUser(t *testing.T, db *gorm.DB, name string, role models.Role) models.User {
	t.Helper()
	user := models.User{
		ID: uuid.New(), Name: name, Email: uuid.NewString() + "@test.local",
		Password: "hashed", Role: role, IsActive: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func mustEngineer(t *testing.T, db *gorm.DB, userID uuid.UUID) {
	t.Helper()
	row := models.SupportEngineer{ID: uuid.New(), UserID: userID, Designation: "Support Engineer", IsActive: true}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
}

func installInput(siteID uuid.UUID) CreateTourAdvanceInput {
	return CreateTourAdvanceInput{
		CustomerID: siteID, WorkType: "INSTALLATION", PONumber: "THPG26/263410240",
		PersonsTravelling: 3, DaysPlanned: 3,
		Travellers:  []TourTravellerInput{{Name: "One"}, {Name: "Two"}, {Name: "Three"}},
		FoodExpense: "2700", LocalExpense: "1000", AccommodationExpense: "4500",
		TravelMode: "CAR", TravelExpense: "2000",
	}
}

func TestTourAdvanceCreateValidation(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)

	_, err := svc.Create(fx.engineer.ID, models.RoleSupport, installInput(uuid.New()))
	assertRule(t, err, "customer not found")

	missingPO := installInput(fx.siteA.ID)
	missingPO.PONumber = ""
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, missingPO)
	assertRule(t, err, "PO number is required for installation")

	withTicket := installInput(fx.siteA.ID)
	withTicket.TicketID = fx.ticketA.ID
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, withTicket)
	assertRule(t, err, "installation cannot include a ticket")

	serviceMissing := installInput(fx.siteA.ID)
	serviceMissing.WorkType = "SERVICE"
	serviceMissing.PONumber = ""
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, serviceMissing)
	assertRule(t, err, "ticket is required for service")

	servicePO := serviceMissing
	servicePO.TicketID = fx.ticketA.ID
	servicePO.PONumber = "PO-9"
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, servicePO)
	assertRule(t, err, "service cannot include a PO number")

	wrongTicket := serviceMissing
	wrongTicket.TicketID = fx.ticketB.ID
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, wrongTicket)
	assertRule(t, err, "ticket does not belong to the selected customer")

	closedTicket := serviceMissing
	closedTicket.TicketID = "VS/10/26/3"
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, closedTicket)
	assertRule(t, err, "closed tickets cannot be selected")

	badPersons := installInput(fx.siteA.ID)
	badPersons.PersonsTravelling = 0
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, badPersons)
	assertRule(t, err, "persons travelling must be greater than 0")

	badDays := installInput(fx.siteA.ID)
	badDays.DaysPlanned = -1
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, badDays)
	assertRule(t, err, "days planned must be greater than 0")

	negative := installInput(fx.siteA.ID)
	negative.FoodExpense = "-5"
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, negative)
	assertRule(t, err, "expenses cannot be negative")

	_, err = svc.Create(fx.admin.ID, models.RoleAdmin, installInput(fx.siteA.ID))
	if !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("admin create: %v", err)
	}

	var engineer models.SupportEngineer
	if err := db.Where("user_id = ?", fx.engineer.ID).First(&engineer).Error; err != nil {
		t.Fatal(err)
	}
	withPeople := installInput(fx.siteA.ID)
	withPeople.PersonsTravelling = 2
	withPeople.Travellers = []TourTravellerInput{
		{EngineerID: engineer.ID.String()},
		{Name: "Site helper"},
	}
	withPeople.TravelFrom = "2026-10-02"
	withPeople.TravelTo = "2026-10-04"
	created, err := svc.Create(fx.engineer.ID, models.RoleSupport, withPeople)
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Travellers) != 2 || created.Travellers[0].Kind != "engineer" || created.Travellers[0].Name != "Sarvesh" {
		t.Fatalf("travellers %#v", created.Travellers)
	}
	if created.Travellers[1].Kind != "other" || created.Travellers[1].Name != "Site helper" {
		t.Fatalf("other traveller %#v", created.Travellers[1])
	}
	if created.TravelFrom == nil || *created.TravelFrom != "2026-10-02" || created.DaysPlanned != 3 {
		t.Fatalf("dates from=%v days=%d", created.TravelFrom, created.DaysPlanned)
	}

	mismatch := installInput(fx.siteA.ID)
	mismatch.Travellers = []TourTravellerInput{{Name: "Only one"}}
	_, err = svc.Create(fx.engineer.ID, models.RoleSupport, mismatch)
	assertRule(t, err, "select every person travelling")
}

func TestTourAdvanceWorkflow(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)

	created, err := svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != string(models.TourStatusPendingApproval) {
		t.Fatalf("status %s", created.Status)
	}
	if created.RequestedAmount != "10200.00" {
		t.Fatalf("requested %s", created.RequestedAmount)
	}
	if created.CustomerName != "TVARUR Oils And Fats Private Limited" || created.Location != "TVARUR" {
		t.Fatalf("customer=%s location=%s", created.CustomerName, created.Location)
	}
	if created.RequestNumber != "TR-000001" {
		t.Fatalf("number %s", created.RequestNumber)
	}

	_, err = svc.Get(fx.otherEngineer.ID, models.RoleSupport, created.ID)
	if !errors.Is(err, ErrTourNotFound) {
		t.Fatalf("other engineer get: %v", err)
	}

	_, err = svc.Approve(fx.engineer.ID, models.RoleSupport, created.ID, "10000", "")
	if !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("support approve: %v", err)
	}
	_, err = svc.Approve(fx.admin.ID, models.RoleAdmin, created.ID, "10000", "")
	if !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("admin approve: %v", err)
	}
	_, err = svc.Approve(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "10200.01", "")
	if err == nil {
		t.Fatal("expected approved amount cap")
	}
	_, err = svc.Reject(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "  ")
	assertRule(t, err, "rejection reason is required")

	approved, err := svc.Approve(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "10000", "rounded")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != string(models.TourStatusApproved) || approved.ApprovedAmount == nil || *approved.ApprovedAmount != "10000.00" {
		t.Fatalf("approved view %+v", approved.ApprovedAmount)
	}
	if approved.RequestedAmount != "10200.00" {
		t.Fatalf("requested overwritten: %s", approved.RequestedAmount)
	}
	if approved.ApprovedByName != "Super Admin" {
		t.Fatalf("approved by %s", approved.ApprovedByName)
	}

	_, err = svc.MarkBillSubmitted(fx.otherEngineer.ID, models.RoleSupport, created.ID)
	if !errors.Is(err, ErrTourNotFound) {
		t.Fatalf("other bill: %v", err)
	}
	_, err = svc.Process(fx.admin.ID, models.RoleAdmin, created.ID)
	if !errors.Is(err, ErrTourNotFound) {
		t.Fatalf("process before bill: %v", err)
	}
	_, err = svc.Process(fx.engineer.ID, models.RoleSupport, created.ID)
	if !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("support process: %v", err)
	}
	_, err = svc.Process(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID)
	if !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("super process: %v", err)
	}

	billed, err := svc.MarkBillSubmitted(fx.engineer.ID, models.RoleSupport, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if billed.Status != string(models.TourStatusBillSubmitted) || billed.BillSubmittedByName != "Sarvesh" {
		t.Fatalf("billed %+v", billed)
	}
	_, err = svc.MarkBillSubmitted(fx.engineer.ID, models.RoleSupport, created.ID)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("repeat bill: %v", err)
	}

	adminList, err := svc.List(fx.admin.ID, models.RoleAdmin, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(adminList.Requests) != 1 || adminList.Requests[0].ID != created.ID {
		t.Fatalf("admin queue %#v", adminList.Requests)
	}

	processed, err := svc.Process(fx.admin.ID, models.RoleAdmin, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if processed.Status != string(models.TourStatusProcessed) {
		t.Fatalf("processed %s", processed.Status)
	}
	_, err = svc.Process(fx.admin.ID, models.RoleAdmin, created.ID)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("repeat process: %v", err)
	}
	_, err = svc.Approve(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "1000", "")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("approve processed: %v", err)
	}
	_, err = svc.MarkBillSubmitted(fx.engineer.ID, models.RoleSupport, created.ID)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("bill processed: %v", err)
	}
}

func TestTourAdvanceRejectAndTicketFilter(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)

	serviceIn := installInput(fx.siteA.ID)
	serviceIn.WorkType = "SERVICE"
	serviceIn.PONumber = ""
	serviceIn.TicketID = fx.ticketA.ID
	created, err := svc.Create(fx.engineer.ID, models.RoleSupport, serviceIn)
	if err != nil {
		t.Fatal(err)
	}
	if created.TicketID == nil || *created.TicketID != fx.ticketA.ID || created.PONumber != nil {
		t.Fatalf("service fields po=%v ticket=%v", created.PONumber, created.TicketID)
	}

	rejected, err := svc.Reject(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "PO information needs correction")
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != string(models.TourStatusRejected) {
		t.Fatalf("status %s", rejected.Status)
	}
	_, err = svc.Approve(fx.superAdmin.ID, models.RoleSuperAdmin, created.ID, "1000", "")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("approve rejected: %v", err)
	}
	_, err = svc.MarkBillSubmitted(fx.engineer.ID, models.RoleSupport, created.ID)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("bill rejected: %v", err)
	}

	tickets, err := svc.ListTickets(fx.siteA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 || tickets[0].ID != fx.ticketA.ID {
		t.Fatalf("tickets %#v", tickets)
	}

	stored := models.TourAdvanceRequest{}
	if err := db.First(&stored, "id = ?", created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !stored.RequestedAmount.Equal(decimal.RequireFromString("10200")) {
		t.Fatalf("stored total %s", stored.RequestedAmount)
	}
}

func TestTourAdvanceDeleteRemovesRequestEventsAndNotifications(t *testing.T) {
	db := openTourDB(t)
	fx := seedTour(t, db)
	svc := NewTourAdvanceService(db, nil)

	created, err := svc.Create(fx.engineer.ID, models.RoleSupport, installInput(fx.siteA.ID))
	if err != nil {
		t.Fatal(err)
	}
	id := created.ID
	noteID := uuid.New()
	if err := db.Exec(
		`INSERT INTO notifications (id, user_id, type, title, message, metadata, is_read, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		noteID.String(), fx.superAdmin.ID.String(),
		string(models.NotificationTypeTourAdvanceSubmitted),
		"New tour advance", "TR submitted",
		`{"tour_advance_id":"`+id.String()+`","request_number":"`+created.RequestNumber+`"}`,
	).Error; err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(fx.otherEngineer.ID, models.RoleSupport, id); !errors.Is(err, ErrTourNotFound) {
		t.Fatalf("other engineer: %v", err)
	}
	if err := svc.Delete(fx.admin.ID, models.RoleAdmin, id); !errors.Is(err, ErrTourForbidden) {
		t.Fatalf("admin: %v", err)
	}
	if err := svc.Delete(fx.engineer.ID, models.RoleSupport, id); err != nil {
		t.Fatal(err)
	}

	if err := db.First(&models.TourAdvanceRequest{}, "id = ?", id).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("request still present: %v", err)
	}
	var events int64
	if err := db.Model(&models.TourAdvanceEvent{}).Where("tour_advance_request_id = ?", id).Count(&events).Error; err != nil {
		t.Fatal(err)
	}
	if events != 0 {
		t.Fatalf("events left: %d", events)
	}
	var notes int64
	if err := db.Model(&models.Notification{}).Where("id = ?", noteID).Count(&notes).Error; err != nil {
		t.Fatal(err)
	}
	if notes != 0 {
		t.Fatalf("notifications left: %d", notes)
	}
	if _, err := svc.Get(fx.engineer.ID, models.RoleSupport, id); !errors.Is(err, ErrTourNotFound) {
		t.Fatalf("get after delete: %v", err)
	}
}

const tourTestSchema = `
CREATE TABLE users (
  id text primary key,
  name text,
  email text not null,
  password text not null,
  role text not null,
  is_active numeric,
  must_reset_password numeric,
  created_by text,
  avatar_url text,
  two_fa_enabled numeric,
  last_login_at datetime,
  last_otp_verified_at datetime,
  last_password_reset_at datetime,
  created_at datetime,
  updated_at datetime
);
CREATE TABLE companies (
  id text primary key,
  name text not null,
  is_active numeric,
  created_at datetime,
  updated_at datetime
);
CREATE TABLE customers (
  id text primary key,
  user_id text,
  company_id text,
  name text,
  address text,
  location text,
  plant text,
  phone text,
  email text,
  contact_person text,
  is_active numeric,
  created_at datetime,
  updated_at datetime
);
CREATE TABLE support_engineers (
  id text primary key,
  user_id text,
  designation text,
  phone text,
  is_active numeric,
  created_at datetime,
  updated_at datetime
);
CREATE TABLE tickets (
  id text primary key,
  customer_id text,
  customer_solution_id text,
  asset_id text,
  engineer_id text,
  title text,
  description text,
  status text,
  priority text,
  support_mode text,
  service_call_type text,
  sla_hours integer,
  target_at datetime,
  sla_paused_at datetime,
  sla_paused_total_seconds integer,
  closure_proof_image text,
  support_comment text,
  closed_at datetime,
  created_by text,
  created_at datetime,
  updated_at datetime
);
CREATE TABLE tour_advance_requests (
  id text primary key,
  request_number text,
  engineer_id text,
  customer_id text,
  location_snapshot text,
  work_type text,
  po_number text,
  ticket_id text,
  persons_travelling integer,
  travellers text,
  days_planned integer,
  travel_from text,
  travel_to text,
  food_expense numeric,
  local_expense numeric,
  accommodation_expense numeric,
  travel_mode text,
  travel_expense numeric,
  requested_amount numeric,
  approved_amount numeric,
  status text,
  approval_remarks text,
  approved_by text,
  approved_at datetime,
  rejection_remarks text,
  rejected_by text,
  rejected_at datetime,
  bill_submitted_by text,
  bill_submitted_at datetime,
  processed_by text,
  processed_at datetime,
  created_at datetime,
  updated_at datetime
);
CREATE TABLE tour_advance_events (
  id text primary key,
  tour_advance_request_id text,
  action text,
  actor_user_id text,
  note text,
  created_at datetime
);
CREATE TABLE notifications (
  id text primary key,
  user_id text,
  type text,
  title text,
  message text,
  metadata text,
  is_read integer,
  created_at datetime,
  updated_at datetime
);
`

func assertRule(t *testing.T, err error, msg string) {
	t.Helper()
	var rule *domain.RuleError
	if !errors.As(err, &rule) || rule.Error() != msg {
		t.Fatalf("got %v want %s", err, msg)
	}
}
