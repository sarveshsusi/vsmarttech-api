package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

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

func main() {
	/* =========================
	   ENV & CONFIG
	========================= */
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system env vars")
	}

	cfg := config.LoadConfig()

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	/* =========================
	   DATABASE
	========================= */
	if err := database.Init(cfg); err != nil {
		log.Fatalf("❌ database init failed: %v", err)
	}

	database.Migrate(database.DB)
	db := database.DB

	/* =========================
	   GIN ENGINE
	========================= */
	r := gin.New()

	r.Use(
		gin.Logger(),
		middleware.SafeRecovery(),
	)

	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.AuditLog())
	r.Use(middleware.MaxBodySize(1 << 20)) // 1MB
	// r.Use(middleware.RateLimit(cfg.Server.RateLimitMax))

	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}

	// Set CORS allowed origins - allow both dev and production
	corsOrigins := []string{
		"http://localhost:5173",  // Development
		cfg.FrontendURL,           // Production (from env)
	}

	r.Use(middleware.CORSMiddleware(corsOrigins))

	/* =========================
	   REPOSITORIES
	========================= */
	authRepo := repository.NewAuthRepository(db)
	rememberedDeviceRepo := repository.NewRememberedDeviceRepo(db)

	companyRepo := repository.NewCompanyRepository(db)

	customerRepo := repository.NewCustomerRepository(db)
	supportEngineerRepo := repository.NewSupportEngineerRepository(db)

	solutionRepo := repository.NewSolutionRepository(db)
	customerSolutionRepo := repository.NewCustomerSolutionRepository(db)

	ticketRepo := repository.NewTicketRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	userRepo := repository.NewUserRepository(db)

	/* =========================
	   SERVICES
	========================= */
	authService := service.NewAuthService(
		db,
		authRepo,
		rememberedDeviceRepo,
		customerRepo,
		cfg,
	)

	companyService := service.NewCompanyService(companyRepo)

	customerService := service.NewCustomerService(
		db,
		authRepo,
		customerRepo,
		ticketRepo,
	)

	solutionService := service.NewSolutionService(solutionRepo)

	customerSolutionService := service.NewCustomerSolutionService(
		db,
		customerSolutionRepo,
		customerRepo,
	)

	// Create mailer for notifications and contract expiry
	mailer := utils.NewMailer(cfg.Mail)

	// Create NotificationService with mailer (needed by TicketService)
	notificationService := service.NewNotificationService(
		db,
		notificationRepo,
		ticketRepo,
		userRepo,
		customerRepo,
		mailer,
	)

	ticketService := service.NewTicketService(
		db,
		ticketRepo,
		customerRepo,
		customerSolutionRepo,
		notificationService,
	)

	supportService := service.NewSupportService(
		ticketRepo,
		supportEngineerRepo,
	)

	adminService := service.NewAdminService(dashboardRepo)
	feedbackService := service.NewFeedbackService(feedbackRepo)

	supportEngineerService := service.NewSupportEngineerService(
		supportEngineerRepo,
	)

	// Contract Expiry Service
	// Contract Expiry Service
	dashboardURL := cfg.FrontendURL
	if dashboardURL == "" {
		if cfg.Server.Env == "production" {
			dashboardURL = "https://yourdomain.com" // Change to your production domain
		} else {
			dashboardURL = "http://localhost:5173" // Development default
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

	// AMC Assignment Service
	amcRepo := repository.NewAMCAssignmentRepository(db)
	amcService := service.NewAMCAssignmentService(
		amcRepo,
		notificationService,
		customerSolutionRepo,
	)

	/* =========================
	   UTILS
	========================= */
	var imageUploader utils.ImageUploader
	var err error

	// ✅ ONLY USE AWS S3 - ImageKit is deprecated
	log.Println("✅ Using AWS S3 for file uploads")
	imageUploader, err = utils.NewS3Uploader(cfg)
	if err != nil {
		log.Fatalf("❌ S3 initialization error: %v", err)
	}

	/* =========================
	   HANDLERS
	========================= */
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

	supportEngineerHandler := handler.NewSupportEngineerHandler(
		supportEngineerService,
		supportService,
	)

	notificationHandler := handler.NewNotificationHandler(notificationService)

	contractHandler := handler.NewContractHandler(contractExpiryService)

	amcHandler := handler.NewAMCAssignmentHandler(amcService, imageUploader, supportEngineerRepo)

	/* =========================
	   START CRON JOBS
	========================= */
	jobs.StartContractExpiryCron(contractExpiryService)

	// SLA Escalation Cron Job
	slaEscalationCron := jobs.NewSLAEscalationCron(db, mailer)
	if err := slaEscalationCron.Start(); err != nil {
		log.Printf("⚠️  Failed to start SLA escalation cron: %v", err)
	} else {
		log.Println("✅ SLA escalation cron job started successfully")
	}

	/* =========================
	   ROUTES
	========================= */
	routes.SetupRoutes(
		r,
		cfg,

		// Auth
		authHandler,

		// Dashboards
		adminDashboard,
		supportDashboard,
		customerDashboard,

		// Companies (ADMIN)
		companyHandler,

		// Customers (ADMIN)
		customerHandler,

		// Core
		ticketHandler,
		feedbackHandler,

		// Solutions
		solutionHandler,
		customerSolutionHandler,

		// Support
		supportEngineerHandler,

		// Notifications
		notificationHandler,

		// Contracts (AMC/Warranty)
		contractHandler,

		// AMC Scheduler
		amcHandler,
	)

	/* =========================
	   START SERVER
	========================= */
	log.Printf(
		"🚀 Server running on port %s [%s]",
		cfg.Server.Port,
		cfg.Server.Env,
	)

	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal(err)
	}
}
