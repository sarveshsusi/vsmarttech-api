package tickets

import (
	"github.com/gin-gonic/gin"

	"rbac/handler"
)

// Handlers groups ticket-domain HTTP handlers.
type Handlers struct {
	Ticket   *handler.TicketHandler
	Feedback *handler.FeedbackHandler
}

// RegisterCommon mounts upload helpers shared across roles.
func RegisterCommon(protected *gin.RouterGroup, h Handlers) {
	protected.GET("/imagekit/auth", h.Ticket.GetImageKitAuthToken)
	protected.POST("/upload/proof", h.Ticket.UploadProofImage)
}

// RegisterAdmin mounts admin ticket management routes.
func RegisterAdmin(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/tickets", h.Ticket.GetAdminTickets)
	admin.POST("/tickets", h.Ticket.AdminCreateTicketAndAssign)
	admin.POST("/tickets/assign", h.Ticket.AssignTicket)
	admin.POST("/tickets/reassign", h.Ticket.ReassignTicket)
	admin.POST("/tickets/close", h.Ticket.AdminCloseTicket)
}

// RegisterSupport mounts support ticket lifecycle routes.
func RegisterSupport(support *gin.RouterGroup, h Handlers) {
	support.POST("/tickets/start/*id", h.Ticket.StartTicket)
	support.POST("/tickets/close", h.Ticket.CloseTicket)
}

// RegisterCustomer mounts customer ticket + feedback routes.
func RegisterCustomer(customer *gin.RouterGroup, h Handlers) {
	customer.GET("/ticket", h.Ticket.GetTicketById)
	customer.POST("/tickets", h.Ticket.CreateTicket)
	customer.POST("/tickets/feedback", h.Feedback.Submit)
}
