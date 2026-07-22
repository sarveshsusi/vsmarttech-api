// handler/admin_dashboard.go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"rbac/service"

	"github.com/gin-gonic/gin"
)

type AdminDashboardHandler struct {
	service *service.AdminService
}

func NewAdminDashboardHandler(s *service.AdminService) *AdminDashboardHandler {
	return &AdminDashboardHandler{service: s}
}

func (h *AdminDashboardHandler) Dashboard(c *gin.Context) {
	stats, err := h.service.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dashboard"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *AdminDashboardHandler) OpsKPIs(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "90"))
	kpis, err := h.service.GetOpsKPIs(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ops KPIs"})
		return
	}
	c.JSON(http.StatusOK, kpis)
}

func (h *AdminDashboardHandler) GetDashboardTickets(c *gin.Context) {
	// Query parameters
	company := c.Query("company")
	contractType := c.Query("contract_type")
	status := c.Query("status")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	// Parse pagination
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Prepare filters
	var companyPtr *string
	var contractTypePtr *string
	var statusPtr *string
	var startDate *time.Time
	var endDate *time.Time

	if company != "" {
		companyPtr = &company
	}
	if contractType != "" {
		contractTypePtr = &contractType
	}
	if status != "" {
		statusPtr = &status
	}

	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &t
		}
	}

	if endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Set to end of day
			t = t.Add(time.Hour * 23).Add(time.Minute * 59).Add(time.Second * 59)
			endDate = &t
		}
	}

	// Get tickets
	tickets, total, err := h.service.GetDashboardTickets(
		companyPtr,
		contractTypePtr,
		statusPtr,
		startDate,
		endDate,
		limit,
		offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tickets": tickets,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}
