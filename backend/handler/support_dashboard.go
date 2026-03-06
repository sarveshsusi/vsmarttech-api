package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/service"
)

type SupportDashboardHandler struct {
	service      *service.SupportService
	adminService *service.AdminService
}

func NewSupportDashboardHandler(s *service.SupportService, a *service.AdminService) *SupportDashboardHandler {
	return &SupportDashboardHandler{
		service:      s,
		adminService: a,
	}
}

func (h *SupportDashboardHandler) GetStats(c *gin.Context) {
	engineerID := c.MustGet("user_id").(uuid.UUID)

	stats, err := h.service.GetEngineerDashboardStats(engineerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *SupportDashboardHandler) MyTickets(c *gin.Context) {
	engineerID := c.MustGet("user_id").(uuid.UUID)

	tickets, err := h.service.GetAssignedTickets(engineerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tickets"})
		return
	}

	c.JSON(http.StatusOK, tickets)
}
