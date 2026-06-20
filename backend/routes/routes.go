package routes

import (
	"github.com/gin-gonic/gin"

	"rbac/config"
	"rbac/handler"
	"rbac/middleware"
	"rbac/models"
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

	/* =========================
	   SECURITY MIDDLEWARE (Phase 1)
	========================= */
	// Enhanced audit logging with RequestID tracking
	r.Use(middleware.SecurityAuditMiddleware())

	// Security headers (CSP, X-Frame-Options, CORS validation)
	r.Use(middleware.SecurityHeadersMiddleware(cfg.FrontendURL))
	r.Use(middleware.CORSSecureMiddleware([]string{cfg.FrontendURL}))

	/* =========================
	   STATIC FILES (LOCAL UPLOADS)
	========================= */
	if cfg.Storage.Type == "local" {
		r.Static("/uploads", cfg.Storage.LocalDir)
	}

	api := r.Group("/api/v1")

	/* =========================
	   AUTH (PUBLIC)
	========================= */
	auth := api.Group("/auth")
	// Rate limiting for auth endpoints (10 requests per minute)
	auth.POST("/login", middleware.RateLimit(10), middleware.BruteForceGuard(), authHandler.Login)
	{
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)

		auth.POST(
			"/verify-2fa",
			middleware.Temp2FAMiddleware(cfg),
			authHandler.Verify2FA,
		)
	}

	/* =========================
	   PROTECTED (JWT)
	========================= */
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg))
	// Rate limiting for protected endpoints (60 requests per minute for standard API)
	protected.Use(middleware.RateLimit(60))
	{
		/* ---------- COMMON ---------- */
		protected.POST("/logout", authHandler.Logout)
		protected.GET("/profile", authHandler.GetMe)
		protected.POST("/change-password", authHandler.ChangePassword)

		protected.POST("/2fa/enable", authHandler.Enable2FA)
		protected.POST("/2fa/disable", authHandler.Disable2FA)

		/* =========================
		   IMAGEKIT AUTH (For client-side uploads)
		========================= */
		protected.GET("/imagekit/auth", ticketHandler.GetImageKitAuthToken)

		/* =========================
		   FILE UPLOAD (S3)
		========================= */
		protected.POST("/upload/proof", ticketHandler.UploadProofImage)

		/* =========================
		   NOTIFICATIONS (All users)
		========================= */
		notifications := protected.Group("/notifications")
		{
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.GET("/unread", notificationHandler.GetUnreadCount)
			notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
			notifications.PUT("/all/read", notificationHandler.MarkAllAsRead)
			notifications.GET("/preferences", notificationHandler.GetPreferences)
			notifications.PUT("/preferences", notificationHandler.UpdatePreferences)
			notifications.POST("/test", notificationHandler.TestCreateNotification)

		}

		/* =========================
		   ADMIN
		========================= */
		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRole(models.RoleAdmin))
		// Rate limiting for admin endpoints (30 requests per minute for sensitive operations)
		admin.Use(middleware.RateLimit(30))
		{
			admin.GET("/dashboard", adminDashboard.Dashboard)
			admin.GET("/dashboard/tickets", adminDashboard.GetDashboardTickets)

			/* ---------- USERS ---------- */
			admin.POST("/users", authHandler.CreateUser)
			admin.GET("/users", authHandler.GetAllUsers)
			admin.PUT("/users/:id", authHandler.EditUser)
			admin.DELETE("/users/:id", authHandler.DeleteUser)
			admin.POST("/companies", companyHandler.CreateCompany)
			admin.GET("/companies", companyHandler.GetCompanies)
			admin.PUT("/companies/:id", companyHandler.EditCompany)
			admin.DELETE("/companies/:id", companyHandler.DeleteCompany)
			admin.GET("/customers", customerHandler.GetAll) /* ---------- SUPPORT ENGINEERS ---------- */
			admin.GET(
				"/support-engineers",
				supportEngineerHandler.GetAll,
			)

			/* =========================
			   SOLUTIONS
			========================= */
			admin.POST("/solutions", solutionHandler.Create)
			admin.GET("/solutions", solutionHandler.GetAll)
			admin.PUT("/solutions/:id", solutionHandler.Edit)
			admin.DELETE("/solutions/:id", solutionHandler.Delete)

			/* =========================
			   CUSTOMER → SOLUTION (PO)
			========================= */
			admin.POST(
				"/customer-solutions",
				customerSolutionHandler.AssignSolution,
			)

			admin.GET(
				"/customers/:id/solutions",
				customerSolutionHandler.GetCustomerSolutions,
			)

			admin.GET(
				"/customer-solutions",
				customerSolutionHandler.GetAllCustomerSolutions,
			)

			/* =========================
			   CONTRACTS (AMC/WARRANTY)
			========================= */
			admin.GET("/contracts/amc", contractHandler.GetAMCContracts)
			admin.GET("/contracts/warranty", contractHandler.GetWarrantyContracts)
			admin.POST("/contracts/check-expiry", contractHandler.TriggerExpiryCheck)

			/* =========================
			   AMC SCHEDULER
			========================= */
			admin.POST("/amc-assignments", amcHandler.AssignAMC)
			admin.GET("/amc-assignments", amcHandler.GetAllAMCs)
			admin.GET("/amc-assignments/:id", amcHandler.GetAMCAssignment)
			admin.GET("/amc-assignments/:id/proofs", amcHandler.GetVisitProofs)

			/* =========================
			   TICKETS
			========================= */
			admin.GET("/tickets", ticketHandler.GetAdminTickets)

			// Admin creates ticket ON BEHALF of customer (PO-based)
			admin.POST("/tickets", ticketHandler.AdminCreateTicketAndAssign)

			admin.POST(
				"/tickets/assign",
				ticketHandler.AssignTicket,
			)

			admin.POST(
				"/tickets/reassign",
				ticketHandler.ReassignTicket,
			)

			// Admin closes ticket on behalf of support engineer
			admin.POST(
				"/tickets/close",
				ticketHandler.AdminCloseTicket,
			)
		}

		/* =========================
		   SUPPORT
		========================= */
		support := protected.Group("/support")
		support.Use(middleware.RequireRole(models.RoleSupport))
		{
			support.GET("/dashboard", supportDashboard.GetStats)
			support.GET("/tickets", supportEngineerHandler.GetMyTickets)

			support.POST("/tickets/start", ticketHandler.StartTicket)
			support.POST("/tickets/close", ticketHandler.CloseTicket)

			/* =========================
			   AMC SCHEDULER
			========================= */
			support.GET("/amc-assignments", amcHandler.GetMyAMCs)
			support.GET("/amc-assignments/:id", amcHandler.GetAMCAssignment)
			support.PUT("/amc-visits/:visit_id/complete", amcHandler.CompleteVisit)
			support.POST("/amc-visits/:visit_id/proofs", amcHandler.UploadProof)
			support.GET("/amc-visits/:visit_id/proofs", amcHandler.GetVisitProofs)
		}

		/* =========================
		   CUSTOMER
		========================= */
		customer := protected.Group("/customer")
		customer.Use(middleware.RequireRole(models.RoleCustomer))
		{
			customer.GET("/dashboard", customerDashboard.GetStats)
			customer.GET("/dashboard/checkpoints", customerDashboard.GetTicketCheckpoints)
			customer.GET("/tickets", customerDashboard.MyTickets)
			customer.GET("/ticket", ticketHandler.GetTicketById) // Use query param: ?id=VS/04/26/1

			// Customer creates ticket using PO
			customer.POST("/tickets", ticketHandler.CreateTicket)

			customer.POST(
				"/tickets/feedback",
				feedbackHandler.Submit,
			)

			customer.GET(
				"/solutions",
				customerSolutionHandler.GetMySolutions,
			)
		}
	}
}
