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
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		TicketID string `json:"ticket_id" binding:"required"`
		Rating   int    `json:"rating" binding:"required"`
		Comment  string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.Submit(
		userID,
		req.TicketID,
		req.Rating,
		req.Comment,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "feedback submitted"})
}
