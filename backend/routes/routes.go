package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rbac/config"
	"rbac/handler"
	"rbac/middleware"
	"rbac/models"

	modamc "rbac/internal/modules/amc"
	modauth "rbac/internal/modules/auth"
	modcrm "rbac/internal/modules/crm"
	modnotify "rbac/internal/modules/notify"
	modtickets "rbac/internal/modules/tickets"
)

func SetupRoutes(
	r *gin.Engine,
	cfg *config.Config,
	db *gorm.DB,

	authHandler *handler.AuthHandler,

	adminDashboard *handler.AdminDashboardHandler,
	supportDashboard *handler.SupportDashboardHandler,
	customerDashboard *handler.CustomerDashboardHandler,

	companyHandler *handler.CompanyHandler,
	customerHandler *handler.CustomerHandler,

	ticketHandler *handler.TicketHandler,
	feedbackHandler *handler.FeedbackHandler,

	solutionHandler *handler.SolutionHandler,
	customerSolutionHandler *handler.CustomerSolutionHandler,

	supportEngineerHandler *handler.SupportEngineerHandler,
	notificationHandler *handler.NotificationHandler,
	contractHandler *handler.ContractHandler,
	amcHandler *handler.AMCAssignmentHandler,
	auditHandler *handler.AuditHandler,
	assetHandler *handler.AssetHandler,
) {
	// Security middleware is applied once in bootstrap (CORS + headers + audit).

	if cfg.Storage.Type == "local" {
		r.Static("/uploads", cfg.Storage.LocalDir)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/readyz", func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "db_nil"})
			return
		}
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": err.Error()})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "reason": "db_ping_failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	api := r.Group("/api/v1")

	modauth.RegisterPublic(api, authHandler, cfg)

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg, db))
	rateLimit := cfg.Server.RateLimitMax
	if rateLimit <= 0 {
		rateLimit = 60
	}
	protected.Use(middleware.RateLimit(rateLimit))
	{
		modauth.RegisterProtected(protected, authHandler)

		ticketHandlers := modtickets.Handlers{Ticket: ticketHandler, Feedback: feedbackHandler}
		modtickets.RegisterCommon(protected, ticketHandlers)
		modnotify.Register(protected, notificationHandler)

		crmHandlers := modcrm.Handlers{
			AdminDashboard:    adminDashboard,
			SupportDashboard:  supportDashboard,
			CustomerDashboard: customerDashboard,
			Company:           companyHandler,
			Customer:          customerHandler,
			Solution:          solutionHandler,
			CustomerSolution:  customerSolutionHandler,
			SupportEngineer:   supportEngineerHandler,
			Asset:             assetHandler,
		}
		amcHandlers := modamc.Handlers{Contract: contractHandler, AMC: amcHandler}

		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRole(models.RoleAdmin))
		adminLimit := rateLimit / 2
		if adminLimit < 10 {
			adminLimit = 10
		}
		admin.Use(middleware.RateLimit(adminLimit))
		{
			modauth.RegisterAdminUsers(admin, authHandler)
			modcrm.RegisterAdmin(admin, crmHandlers)
			modamc.RegisterAdmin(admin, amcHandlers)
			modtickets.RegisterAdmin(admin, ticketHandlers)
			admin.GET("/audit-logs", auditHandler.List)
		}

		support := protected.Group("/support")
		support.Use(middleware.RequireRole(models.RoleSupport))
		{
			modcrm.RegisterSupport(support, crmHandlers)
			modtickets.RegisterSupport(support, ticketHandlers)
			modamc.RegisterSupport(support, amcHandlers)
		}

		customer := protected.Group("/customer")
		customer.Use(middleware.RequireRole(models.RoleCustomer))
		{
			modcrm.RegisterCustomer(customer, crmHandlers)
			modtickets.RegisterCustomer(customer, ticketHandlers)
		}
	}
}
