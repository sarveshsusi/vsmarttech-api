package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/models"
	"rbac/repository"
)

type AuditHandler struct {
	repo *repository.AuditRepository
}

func NewAuditHandler(repo *repository.AuditRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

type auditRowResponse struct {
	models.AuditLog
	ActorName  string `json:"actor_name"`
	ActorEmail string `json:"actor_email"`
}

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}

	filter := repository.AuditListFilter{
		Search:    c.Query("search"),
		UserID:    c.Query("user_id"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Limit:     pageSize,
		Offset:    (page - 1) * pageSize,
	}

	rows, total, err := h.repo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load audit log"})
		return
	}

	// Enrich with actor names
	userIDs := make([]uuid.UUID, 0)
	seen := map[uuid.UUID]bool{}
	for _, row := range rows {
		if row.PerformedBy != uuid.Nil && !seen[row.PerformedBy] {
			seen[row.PerformedBy] = true
			userIDs = append(userIDs, row.PerformedBy)
		}
	}

	usersByID := map[uuid.UUID]models.User{}
	if len(userIDs) > 0 {
		var users []models.User
		_ = h.repo.DB().Where("id IN ?", userIDs).Find(&users).Error
		for _, u := range users {
			usersByID[u.ID] = u
		}
	}

	out := make([]auditRowResponse, 0, len(rows))
	for _, row := range rows {
		item := auditRowResponse{AuditLog: row}
		if u, ok := usersByID[row.PerformedBy]; ok {
			item.ActorName = u.Name
			item.ActorEmail = u.Email
		}
		out = append(out, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      out,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
