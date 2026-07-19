package bootstrap

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rbac/config"
	"rbac/database"
	"rbac/handler"
	"rbac/jobs"
	"rbac/middleware"
	"rbac/repository"
	"rbac/routes"
	"rbac/service"
	"rbac/utils"
)

// App holds wired dependencies for API or worker processes.
type App struct {
	Config                *config.Config
	DB                    *gorm.DB
	Engine                *gin.Engine
	ContractExpiryService *service.ContractExpiryService
	SLAEscalationCron     *jobs.SLAEscalationCron
	Mailer                *utils.Mailer
}

// Options controls process bootstrap behavior.
type Options struct {
	EnableHTTP bool
	Migrate    bool
}

// New loads config, opens DB, and wires services.
func New(opts Options) (*App, error) {
	cfg := config.LoadConfig()

	if opts.EnableHTTP && cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := database.Init(cfg); err != nil {
		return nil, err
	}

	if opts.Migrate {
		database.Migrate(database.DB)
	}

	db := database.DB
	mailer := utils.NewMailer(cfg.Mail)

	authRepo := repository.NewAuthRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	customerSolutionRepo := repository.NewCustomerSolutionRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)

	dashboardURL := cfg.FrontendURL
	if dashboardURL == "" {
		if cfg.Server.Env == "production" {
			dashboardURL = "https://yourdomain.com"
		} else {
			dashboardURL = "http://localhost:5173"
		}
	}

	contractExpiryService := service.NewContractExpiryService(
		db,
		customerSolutionRepo,
		customerRepo,
		authRepo,
		notificationRepo,
		mailer,
		dashboardURL,
	)

	app := &App{
		Config:                cfg,
		DB:                    db,
		ContractExpiryService: contractExpiryService,
		Mailer:                mailer,
		SLAEscalationCron:     jobs.NewSLAEscalationCron(db, mailer),
	}

	if opts.EnableHTTP {
		if err := wireHTTP(app, db, mailer, cfg, authRepo, customerRepo, customerSolutionRepo, notificationRepo); err != nil {
			return nil, err
		}
	}

	if cfg.Server.RunInProcessCrons {
		StartInProcessCrons(app)
	}

	return app, nil
}

func wireHTTP(
	app *App,
	db *gorm.DB,
	mailer *utils.Mailer,
	cfg *config.Config,
	authRepo *repository.AuthRepository,
	customerRepo *repository.CustomerRepository,
	customerSolutionRepo *repository.CustomerSolutionRepository,
	notificationRepo *repository.NotificationRepository,
) error {
	rememberedDeviceRepo := repository.NewRememberedDeviceRepo(db)
	companyRepo := repository.NewCompanyRepository(db)
	supportEngineerRepo := repository.NewSupportEngineerRepository(db)
	solutionRepo := repository.NewSolutionRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)
	userRepo := repository.NewUserRepository(db)
	amcRepo := repository.NewAMCAssignmentRepository(db)

	authService := service.NewAuthService(db, authRepo, rememberedDeviceRepo, customerRepo, cfg)
	companyService := service.NewCompanyService(companyRepo)
	customerService := service.NewCustomerService(db, authRepo, customerRepo, ticketRepo)
	solutionService := service.NewSolutionService(solutionRepo)
	customerSolutionService := service.NewCustomerSolutionService(db, customerSolutionRepo, customerRepo)
	notificationService := service.NewNotificationService(db, notificationRepo, ticketRepo, userRepo, customerRepo, mailer)
	ticketService := service.NewTicketService(db, ticketRepo, customerRepo, customerSolutionRepo, notificationService)
	supportService := service.NewSupportService(ticketRepo, supportEngineerRepo)
	adminService := service.NewAdminService(dashboardRepo)
	feedbackService := service.NewFeedbackService(feedbackRepo)
	supportEngineerService := service.NewSupportEngineerService(supportEngineerRepo)
	amcService := service.NewAMCAssignmentService(amcRepo, notificationService, customerSolutionRepo)

	imageUploader, err := utils.NewS3Uploader(cfg)
	if err != nil {
		return err
	}

	authHandler := handler.NewAuthHandler(authService, cfg)
	adminDashboard := handler.NewAdminDashboardHandler(adminService)
	supportDashboard := handler.NewSupportDashboardHandler(supportService, adminService)
	customerDashboard := handler.NewCustomerDashboardHandler(customerService, adminService)
	companyHandler := handler.NewCompanyHandler(companyService)
	customerHandler := handler.NewCustomerHandler(customerService)
	solutionHandler := handler.NewSolutionHandler(solutionService)
	customerSolutionHandler := handler.NewCustomerSolutionHandler(customerSolutionService)
	ticketHandler := handler.NewTicketHandler(ticketService, imageUploader)
	feedbackHandler := handler.NewFeedbackHandler(feedbackService)
	supportEngineerHandler := handler.NewSupportEngineerHandler(supportEngineerService, supportService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	contractHandler := handler.NewContractHandler(app.ContractExpiryService)
	amcHandler := handler.NewAMCAssignmentHandler(amcService, imageUploader, supportEngineerRepo)

	r := gin.New()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.StructuredAccessLog())
	r.Use(middleware.SafeRecovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.AuditLog())
	r.Use(middleware.MaxBodySize(1 << 20))

	if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Printf("warning: SetTrustedProxies: %v", err)
	}

	corsOrigins := []string{cfg.FrontendURL}
	if cfg.Server.Env != "production" {
		corsOrigins = append(corsOrigins,
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		)
	}
	r.Use(middleware.CORSMiddleware(corsOrigins))

	routes.SetupRoutes(
		r,
		cfg,
		db,
		authHandler,
		adminDashboard,
		supportDashboard,
		customerDashboard,
		companyHandler,
		customerHandler,
		ticketHandler,
		feedbackHandler,
		solutionHandler,
		customerSolutionHandler,
		supportEngineerHandler,
		notificationHandler,
		contractHandler,
		amcHandler,
	)
	app.Engine = r
	return nil
}

// StartInProcessCrons starts SLA and contract expiry jobs inside the API process.
// Prefer dedicated worker containers in Compose (RUN_INPROCESS_CRONS=false).
func StartInProcessCrons(app *App) {
	jobs.StartContractExpiryCron(app.ContractExpiryService)
	if err := app.SLAEscalationCron.Start(); err != nil {
		log.Printf("failed to start SLA escalation cron: %v", err)
	} else {
		log.Println("SLA escalation cron started in-process")
	}
	log.Println("contract expiry cron started in-process")
}
