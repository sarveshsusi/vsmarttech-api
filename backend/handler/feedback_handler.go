package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/models"
	"rbac/repository"
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
		Remarks  string `json:"remarks"`
		Comment  string `json:"comment"` // legacy alias
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	remarks := req.Remarks
	if remarks == "" {
		remarks = req.Comment
	}

	fb, err := h.service.Submit(userID, req.TicketID, req.Rating, remarks)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "feedback submitted",
		"feedback": fb,
	})
}

func (h *FeedbackHandler) GetByTicket(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(models.Role)
	ticketID := c.Param("ticketId")
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	fb, err := h.service.GetByTicketID(userID, role, ticketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
		return
	}
	c.JSON(http.StatusOK, fb)
}

func (h *FeedbackHandler) GetMine(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	data, err := h.service.GetMyEngineerFeedback(userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *FeedbackHandler) GetByEngineer(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(models.Role)
	engineerID, err := uuid.Parse(c.Param("engineerId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	data, err := h.service.GetEngineerFeedback(userID, role, engineerID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *FeedbackHandler) ListPending(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(models.Role)

	rows, err := h.service.ListPending(userID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *FeedbackHandler) Analytics(c *gin.Context) {
	filter := repository.FeedbackListFilter{}

	if v := c.Query("company_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		filter.CompanyID = &id
	}
	if v := c.Query("engineer_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		filter.EngineerID = &id
	}
	if v := c.Query("customer_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		filter.CustomerID = &id
	}
	if v := c.Query("rating"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		filter.Rating = &n
	}
	if v := c.Query("service_call_type"); v != "" {
		filter.ServiceCallType = &v
	}
	if v := c.Query("priority"); v != "" {
		filter.Priority = &v
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		filter.From = &t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		end := t.Add(24*time.Hour - time.Nanosecond)
		filter.To = &end
	}

	data, err := h.service.Analytics(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to load analytics"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *FeedbackHandler) Reopen(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	fb, err := h.service.Reopen(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "feedback reopened", "feedback": fb})
}
