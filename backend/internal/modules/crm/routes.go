package crm

import (
	"github.com/gin-gonic/gin"

	"rbac/handler"
	"rbac/middleware"
	"rbac/models"
)

// Handlers groups CRM-owned HTTP handlers.
type Handlers struct {
	AdminDashboard    *handler.AdminDashboardHandler
	SupportDashboard  *handler.SupportDashboardHandler
	CustomerDashboard *handler.CustomerDashboardHandler
	Company           *handler.CompanyHandler
	Customer          *handler.CustomerHandler
	Solution          *handler.SolutionHandler
	CustomerSolution  *handler.CustomerSolutionHandler
	SupportEngineer   *handler.SupportEngineerHandler
}

// RegisterAdmin mounts company/customer/solution admin routes.
func RegisterAdmin(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/dashboard", h.AdminDashboard.Dashboard)
	admin.GET("/dashboard/tickets", h.AdminDashboard.GetDashboardTickets)

	admin.POST("/companies", h.Company.CreateCompany)
	admin.GET("/companies", h.Company.GetCompanies)
	admin.PUT("/companies/:id", h.Company.EditCompany)
	admin.DELETE("/companies/:id", h.Company.DeleteCompany)

	admin.GET("/customers", h.Customer.GetAll)
	admin.GET("/support-engineers", h.SupportEngineer.GetAll)

	admin.POST("/solutions", h.Solution.Create)
	admin.GET("/solutions", h.Solution.GetAll)
	admin.PUT("/solutions/:id", h.Solution.Edit)
	admin.DELETE("/solutions/:id", h.Solution.Delete)

	admin.POST("/customer-solutions", h.CustomerSolution.AssignSolution)
	admin.GET("/customers/:id/solutions", h.CustomerSolution.GetCustomerSolutions)
	admin.GET("/customer-solutions", h.CustomerSolution.GetAllCustomerSolutions)
	admin.PUT("/customer-solutions/:id", h.CustomerSolution.UpdateCustomerSolution)
	admin.DELETE("/customer-solutions/:id", h.CustomerSolution.DeleteCustomerSolution)
}

// RegisterSupport mounts support dashboard routes owned by CRM.
func RegisterSupport(support *gin.RouterGroup, h Handlers) {
	support.GET("/dashboard", h.SupportDashboard.GetStats)
	support.GET("/tickets", h.SupportEngineer.GetMyTickets)
	// Peer list for co-engineer selection on field visits (active only)
	support.GET("/engineers", h.SupportEngineer.GetAllActive)
}

// RegisterCustomer mounts customer dashboard + solutions routes.
func RegisterCustomer(customer *gin.RouterGroup, h Handlers) {
	customer.GET("/dashboard", h.CustomerDashboard.GetStats)
	customer.GET("/dashboard/checkpoints", h.CustomerDashboard.GetTicketCheckpoints)
	customer.GET("/tickets", h.CustomerDashboard.MyTickets)
	customer.GET("/solutions", h.CustomerSolution.GetMySolutions)
}

// Role helpers kept next to CRM route groups for discoverability.
func RequireSupport() gin.HandlerFunc {
	return middleware.RequireRole(models.RoleSupport)
}

func RequireCustomer() gin.HandlerFunc {
	return middleware.RequireRole(models.RoleCustomer)
}
