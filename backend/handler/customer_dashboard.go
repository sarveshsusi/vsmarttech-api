package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/service"
)

type CustomerDashboardHandler struct {
	service      *service.CustomerService
	adminService *service.AdminService
}

func NewCustomerDashboardHandler(s *service.CustomerService, a *service.AdminService) *CustomerDashboardHandler {
	return &CustomerDashboardHandler{
		service:      s,
		adminService: a,
	}
}

func (h *CustomerDashboardHandler) GetStats(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// Get customer ID from user
	customer, err := h.service.GetCustomerByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get customer"})
		return
	}

	// Get stats for this customer
	stats, err := h.adminService.GetCustomerDashboardStats(customer.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *CustomerDashboardHandler) GetTicketCheckpoints(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// Get customer ID from user
	customer, err := h.service.GetCustomerByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get customer"})
		return
	}

	// Get ticket checkpoints (status tracking)
	checkpoints, err := h.adminService.GetCustomerTicketCheckpoints(customer.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load checkpoints"})
		return
	}

	c.JSON(http.StatusOK, checkpoints)
}

func (h *CustomerDashboardHandler) MyTickets(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	tickets, err := h.service.GetMyTickets(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, tickets)
}
