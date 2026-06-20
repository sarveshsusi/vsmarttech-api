package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/service"
)

type FeedbackHandler struct {
	service *service.FeedbackService
}

func NewFeedbackHandler(s *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{service: s}
}

func (h *FeedbackHandler) Submit(c *gin.Context) {
	var req struct {
		TicketID   string    `json:"ticket_id" binding:"required"`
		EngineerID uuid.UUID `json:"engineer_id"`
		Rating     int       `json:"rating"`
		Comment    string    `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.Submit(
		req.TicketID,
		req.EngineerID,
		req.Rating,
		req.Comment,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feedback submitted"})
}
