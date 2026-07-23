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
	protected.POST("/upload/proof", h.Ticket.UploadProofImage)
}

// RegisterAdmin mounts admin ticket management routes.
func RegisterAdmin(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/tickets", h.Ticket.GetAdminTickets)
	admin.POST("/tickets", h.Ticket.AdminCreateTicketAndAssign)
	admin.POST("/tickets/assign", h.Ticket.AssignTicket)
	admin.POST("/tickets/reassign", h.Ticket.ReassignTicket)
	admin.POST("/tickets/close", h.Ticket.AdminCloseTicket)
	admin.POST("/tickets/reopen", h.Ticket.ReopenTicket)
	admin.GET("/tickets/status", h.Ticket.ListTicketStatus)
	admin.GET("/tickets/events", h.Ticket.ListTicketEvents)
	admin.GET("/tickets/activity", h.Ticket.ListRecentTicketEvents)
	admin.GET("/tickets/visits", h.Ticket.ListAdminTicketVisits)
	admin.GET("/tickets/comments", h.Ticket.ListTicketComments)
	admin.POST("/tickets/comments", h.Ticket.AddTicketComment)
}

// RegisterSupport mounts support ticket lifecycle routes.
func RegisterSupport(support *gin.RouterGroup, h Handlers) {
	support.POST("/tickets/start/*id", h.Ticket.StartTicket)
	support.POST("/tickets/close", h.Ticket.CloseTicket)
	support.GET("/tickets/visits", h.Ticket.ListSupportTicketVisits)
	support.POST("/tickets/visits", h.Ticket.CreateSupportTicketVisit)
	support.GET("/tickets/comments", h.Ticket.ListTicketComments)
	support.POST("/tickets/comments", h.Ticket.AddTicketComment)
	support.GET("/feedback/me", h.Feedback.GetMine)
	support.GET("/feedback/pending", h.Feedback.ListPending)
}

// RegisterCustomer mounts customer ticket + feedback routes.
func RegisterCustomer(customer *gin.RouterGroup, h Handlers) {
	customer.GET("/ticket", h.Ticket.GetTicketById)
	customer.POST("/tickets", h.Ticket.CreateTicket)
	customer.POST("/tickets/feedback", h.Feedback.Submit)
}

// RegisterFeedback mounts shared feedback read APIs (auth already applied).
func RegisterFeedback(protected *gin.RouterGroup, h Handlers) {
	protected.GET("/feedback/ticket/:ticketId", h.Feedback.GetByTicket)
	protected.GET("/feedback/engineer/:engineerId", h.Feedback.GetByEngineer)
	protected.GET("/feedback/pending", h.Feedback.ListPending)
}

// RegisterAdminFeedback mounts admin-only feedback routes.
func RegisterAdminFeedback(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/feedback/analytics", h.Feedback.Analytics)
	admin.POST("/feedback/:id/reopen", h.Feedback.Reopen)
	admin.GET("/feedback/pending", h.Feedback.ListPending)
	admin.GET("/feedback/engineer/:engineerId", h.Feedback.GetByEngineer)
	admin.GET("/feedback/ticket/:ticketId", h.Feedback.GetByTicket)
}
