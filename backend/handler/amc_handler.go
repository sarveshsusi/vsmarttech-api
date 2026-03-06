// handler/amc_handler.go
package handler

import (
	"net/http"
	"time"

	"rbac/models"
	"rbac/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AMCHandler struct {
	service *service.AMCService
}

func NewAMCHandler(s *service.AMCService) *AMCHandler {
	return &AMCHandler{service: s}
}

// Admin: View all AMC contracts
func (h *AMCHandler) GetAllAMCs(c *gin.Context) {
	amcs, err := h.service.GetAllAMCs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch amcs"})
		return
	}
	c.JSON(http.StatusOK, amcs)
}

// Customer: View own AMC contracts
func (h *AMCHandler) GetMyAMCs(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(models.Role)

	amcs, err := h.service.GetCustomerAMCs(userID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, amcs)
}
func (h *AMCHandler) Create(c *gin.Context) {
	var req struct {
		CustomerProductID uuid.UUID `json:"customer_product_id"`
		SLAHours          int       `json:"sla_hours"`
		StartDate         time.Time `json:"start_date"`
		EndDate           time.Time `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	amc, err := h.service.CreateAMC(
		req.CustomerProductID,
		req.SLAHours,
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(201, amc)
}
