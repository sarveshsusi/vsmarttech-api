package touradvance

import (
	"github.com/gin-gonic/gin"

	"rbac/handler"
	"rbac/middleware"
	"rbac/models"
)

func RegisterSupport(support *gin.RouterGroup, h *handler.TourAdvanceHandler) {
	support.GET("/tour-advances/lookups/companies", h.LookupCompanies)
	support.GET("/tour-advances/lookups/sites", h.LookupSites)
	support.GET("/tour-advances/lookups/tickets", h.LookupTickets)
	support.GET("/tour-advances", h.List)
	support.POST("/tour-advances", h.Create)
	support.GET("/tour-advances/:id", h.Get)
	support.DELETE("/tour-advances/:id", h.Delete)
	support.POST("/tour-advances/:id/bill-submitted", h.BillSubmitted)
}

func RegisterAdmin(admin *gin.RouterGroup, h *handler.TourAdvanceHandler) {
	admin.GET("/tour-advances", h.List)
	admin.GET("/tour-advances/:id", h.Get)

	approve := admin.Group("")
	approve.Use(middleware.RequireRole(models.RoleSuperAdmin))
	approve.POST("/tour-advances/:id/approve", h.Approve)
	approve.POST("/tour-advances/:id/reject", h.Reject)
}

func RegisterProcess(adminOnly *gin.RouterGroup, h *handler.TourAdvanceHandler) {
	adminOnly.POST("/tour-advances/:id/process", h.Process)
}
