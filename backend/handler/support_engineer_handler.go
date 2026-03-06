package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/service"
)

type SupportEngineerHandler struct {
	engineerService *service.SupportEngineerService
	supportService  *service.SupportService
}

/* =========================
   CONSTRUCTOR
========================= */

func NewSupportEngineerHandler(
	engineerService *service.SupportEngineerService,
	supportService *service.SupportService,
) *SupportEngineerHandler {
	return &SupportEngineerHandler{
		engineerService: engineerService,
		supportService:  supportService,
	}
}

/* =========================
   ADMIN: GET ALL ENGINEERS
========================= */

func (h *SupportEngineerHandler) GetAll(c *gin.Context) {
	engineers, err := h.engineerService.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to load support engineers",
		})
		return
	}

	c.JSON(http.StatusOK, engineers)
}

func (h *SupportEngineerHandler) GetAllActive(c *gin.Context) {
	engineers, err := h.engineerService.GetAllActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to load support engineers",
		})
		return
	}

	c.JSON(http.StatusOK, engineers)
}

/* =========================
   SUPPORT: MY TICKETS
========================= */

func (h *SupportEngineerHandler) GetMyTickets(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	tickets, err := h.supportService.GetMyTickets(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to load tickets",
		})
		return
	}

	c.JSON(http.StatusOK, tickets)
}
