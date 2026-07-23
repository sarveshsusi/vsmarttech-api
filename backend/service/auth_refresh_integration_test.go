//go:build integration

package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"rbac/config"
	"rbac/database"
	"rbac/models"
	"rbac/repository"
	"rbac/service"
	"rbac/utils"
)

type errDockerUnavailable struct{ v any }

func (e errDockerUnavailable) Error() string {
	return "docker unavailable"
}

func setupIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		if _, err := os.Stat("/var/run/docker.sock"); err != nil {
			// Docker Desktop on macOS often uses ~/.docker/run/docker.sock
			if _, err2 := os.Stat(os.Getenv("HOME") + "/.docker/run/docker.sock"); err2 != nil {
				t.Skip("Docker not available; set INTEGRATION_DATABASE_URL to run integration tests")
			}
		}
		ctx := context.Background()
		var pg *postgres.PostgresContainer
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = errDockerUnavailable{r}
				}
			}()
			pg, err = postgres.Run(ctx,
				"postgres:15-alpine",
				postgres.WithDatabase("vsmart_test"),
				postgres.WithUsername("test"),
				postgres.WithPassword("test"),
				testcontainers.WithWaitStrategy(
					wait.ForLog("database system is ready to accept connections").
						WithOccurrence(2).
						WithStartupTimeout(60*time.Second),
				),
			)
		}()
		if err != nil {
			t.Skipf("postgres container unavailable (set INTEGRATION_DATABASE_URL): %v", err)
		}
		t.Cleanup(func() {
			_ = pg.Terminate(ctx)
		})
		dsn, err = pg.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			t.Fatalf("connection string: %v", err)
		}
	}

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}

	database.Migrate(db, "auto")
	return db
}

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Env: "test", CookieSameSite: "lax"},
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-integration",
			RefreshSecret: "test-refresh-secret-integration",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
		FrontendURL: "http://localhost:5173",
	}
}

func createUser(t *testing.T, db *gorm.DB, role models.Role, email string) *models.User {
	t.Helper()
	hash, err := utils.HashPassword("Password1!")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	now := time.Now()
	u := &models.User{
		ID:                  uuid.New(),
		Name:                "Test " + string(role),
		Email:               email,
		Password:            hash,
		Role:                role,
		IsActive:            true,
		TwoFAEnabled:        false,
		LastPasswordResetAt: &now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func TestRefreshTokenReuseRevokesFamily(t *testing.T) {
	db := setupIntegrationDB(t)
	cfg := testConfig()
	authRepo := repository.NewAuthRepository(db)
	deviceRepo := repository.NewRememberedDeviceRepo(db)
	customerRepo := repository.NewCustomerRepository(db)
	authSvc := service.NewAuthService(db, authRepo, deviceRepo, customerRepo, cfg)

	user := createUser(t, db, models.RoleAdmin, "admin-refresh@test.local")

	familyID := uuid.New()
	raw1 := "refresh-token-raw-one-" + uuid.NewString()
	if err := authRepo.CreateRefreshToken(&models.RefreshToken{
		UserID:    user.ID,
		FamilyID:  familyID,
		Token:     utils.HashToken(raw1),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("create refresh: %v", err)
	}

	resp, err := authSvc.RefreshAccessToken(raw1, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if resp.RefreshToken == "" || resp.AccessToken == "" {
		t.Fatal("expected new tokens")
	}

	if _, err := authSvc.RefreshAccessToken(raw1, "127.0.0.1", "test"); err == nil {
		t.Fatal("expected reuse to fail")
	}

	if _, err := authSvc.RefreshAccessToken(resp.RefreshToken, "127.0.0.1", "test"); err == nil {
		t.Fatal("expected family-revoked current token to fail")
	}
}

func TestTicketCommentAssigneeOnly(t *testing.T) {
	db := setupIntegrationDB(t)

	admin := createUser(t, db, models.RoleAdmin, "admin-comments@test.local")
	userA := createUser(t, db, models.RoleSupport, "support-a@test.local")
	userB := createUser(t, db, models.RoleSupport, "support-b@test.local")
	custUser := createUser(t, db, models.RoleCustomer, "customer-comments@test.local")

	engA := &models.SupportEngineer{
		ID: uuid.New(), UserID: userA.ID, Designation: "Engineer", IsActive: true,
	}
	engB := &models.SupportEngineer{
		ID: uuid.New(), UserID: userB.ID, Designation: "Engineer", IsActive: true,
	}
	if err := db.Create(engA).Error; err != nil {
		t.Fatalf("engA: %v", err)
	}
	if err := db.Create(engB).Error; err != nil {
		t.Fatalf("engB: %v", err)
	}

	company := &models.Company{ID: uuid.New(), Name: "Test Co Comments", IsActive: true}
	if err := db.Create(company).Error; err != nil {
		t.Fatalf("company: %v", err)
	}

	customer := &models.Customer{
		ID: uuid.New(), UserID: custUser.ID, CompanyID: company.ID,
		Name: "Cust", Email: custUser.Email, IsActive: true,
	}
	if err := db.Create(customer).Error; err != nil {
		t.Fatalf("customer: %v", err)
	}

	ticketID := "VS/07/26/99"
	ticket := &models.Ticket{
		ID:              ticketID,
		CustomerID:      customer.ID,
		EngineerID:      &engA.ID,
		Title:           "Comment auth test",
		Description:     "desc",
		Status:          models.StatusAssigned,
		Priority:        models.PriorityStandard,
		SupportMode:     models.SupportModeRemote,
		ServiceCallType: models.ServiceTypeService,
		CreatedBy:       admin.ID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := db.Create(ticket).Error; err != nil {
		t.Fatalf("ticket: %v", err)
	}

	ticketRepo := repository.NewTicketRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	csRepo := repository.NewCustomerSolutionRepository(db)
	ticketSvc := service.NewTicketService(db, ticketRepo, customerRepo, csRepo, nil)

	if _, err := ticketSvc.AddTicketComment(ticketID, userA.ID, models.RoleSupport, "hello from A", false); err != nil {
		t.Fatalf("assignee add: %v", err)
	}

	if _, err := ticketSvc.AddTicketComment(ticketID, userB.ID, models.RoleSupport, "hello from B", false); err == nil {
		t.Fatal("expected non-assignee add to fail")
	}
	if _, err := ticketSvc.ListTicketComments(ticketID, userB.ID, models.RoleSupport); err == nil {
		t.Fatal("expected non-assignee list to fail")
	}

	comments, err := ticketSvc.ListTicketComments(ticketID, userA.ID, models.RoleSupport)
	if err != nil {
		t.Fatalf("assignee list: %v", err)
	}
	if len(comments) < 1 {
		t.Fatal("expected at least one comment")
	}

	if _, err := ticketSvc.AddTicketComment(ticketID, admin.ID, models.RoleAdmin, "admin note", true); err != nil {
		t.Fatalf("admin add: %v", err)
	}
}
