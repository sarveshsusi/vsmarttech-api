package touradvance

import (
	"github.com/gin-gonic/gin"

	"rbac/handler"
	"rbac/middleware"
	"rbac/models"
)

func RegisterSupport(support *gin.RouterGroup, h *handler.TourAdvanceHandler) {
	support.GET("/tour-advances/eligibility", h.Eligibility)
	support.GET("/tour-advances/history", h.SupportHistory)
	support.GET("/tour-advances/lookups/companies", h.LookupCompanies)
	support.GET("/tour-advances/lookups/sites", h.LookupSites)
	support.GET("/tour-advances/lookups/tickets", h.LookupTickets)
	support.GET("/tour-advances", h.List)
	support.POST("/tour-advances", h.Create)
	support.GET("/tour-advances/:id", h.Get)
	support.DELETE("/tour-advances/:id", h.Delete)
	support.POST("/tour-advances/:id/additional", h.CreateAdditional)
}

func RegisterAdmin(admin *gin.RouterGroup, h *handler.TourAdvanceHandler) {
	admin.GET("/tour-advances/summary", h.Summary)
	admin.GET("/tour-advances/engineers/:engineerId", h.EngineerHistory)
	admin.GET("/tour-advances", h.List)
	admin.GET("/tour-advances/:id", h.Get)

	approve := admin.Group("")
	approve.Use(middleware.RequireRole(models.RoleSuperAdmin))
	approve.POST("/tour-advances/:id/approve", h.Approve)
	approve.POST("/tour-advances/:id/reject", h.Reject)
	approve.POST("/tour-advances/:id/disburse", h.Disburse)
	approve.POST("/tour-advances/:id/additional/approve", h.ApproveAdditional)
	approve.POST("/tour-advances/:id/additional/reject", h.RejectAdditional)
	approve.POST("/tour-advances/:id/additional/disburse", h.DisburseAdditional)
}

func RegisterProcess(adminOnly *gin.RouterGroup, h *handler.TourAdvanceHandler) {
	adminOnly.POST("/tour-advances/:id/process", h.Process)
	adminOnly.POST("/tour-advances/:id/expenses", h.BillSubmitted)
	adminOnly.POST("/tour-advances/:id/expenses/:expenseId", h.AmendExpense)
	adminOnly.POST("/tour-advances/:id/expenses/:expenseId/review", h.ReviewExpense)
	adminOnly.GET("/tour-advances/policy", h.GetPolicy)
	adminOnly.PUT("/tour-advances/policy", h.UpdatePolicy)
}
