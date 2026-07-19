package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"rbac/config"
	"rbac/handler"
	"rbac/middleware"
	"rbac/models"

	modauth "rbac/internal/modules/auth"
	modamc "rbac/internal/modules/amc"
	modcrm "rbac/internal/modules/crm"
	modnotify "rbac/internal/modules/notify"
	modtickets "rbac/internal/modules/tickets"
)

func SetupRoutes(
	r *gin.Engine,
	cfg *config.Config,

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
) {
	// Security middleware is applied once in bootstrap (CORS + headers + audit).
	// Do not re-register CORSSecureMiddleware / SecurityHeadersMiddleware here.

	if cfg.Storage.Type == "local" {
		r.Static("/uploads", cfg.Storage.LocalDir)
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")

	modauth.RegisterPublic(api, authHandler, cfg)

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg))
	protected.Use(middleware.RateLimit(60))
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
		}
		amcHandlers := modamc.Handlers{Contract: contractHandler, AMC: amcHandler}

		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRole(models.RoleAdmin))
		admin.Use(middleware.RateLimit(30))
		{
			modauth.RegisterAdminUsers(admin, authHandler)
			modcrm.RegisterAdmin(admin, crmHandlers)
			modamc.RegisterAdmin(admin, amcHandlers)
			modtickets.RegisterAdmin(admin, ticketHandlers)
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
