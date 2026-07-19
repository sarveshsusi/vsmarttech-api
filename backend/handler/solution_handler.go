package handler

import (
	"rbac/service"
	"rbac/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SolutionHandler struct {
	service *service.SolutionService
}

func NewSolutionHandler(s *service.SolutionService) *SolutionHandler {
	return &SolutionHandler{service: s}
}

func (h *SolutionHandler) Create(c *gin.Context) {
	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	adminID := c.MustGet("user_id").(uuid.UUID)

	solution, err := h.service.Create(req.Title, req.Description, adminID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(201, solution)
}

func (h *SolutionHandler) GetAll(c *gin.Context) {
	list, _ := h.service.GetAll()
	c.JSON(200, list)
}

func (h *SolutionHandler) Edit(c *gin.Context) {
	solutionID := c.Param("id")
	uid, err := uuid.Parse(solutionID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid solution id"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	solution, err := h.service.Update(uid, req.Title, req.Description)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(200, solution)
}

func (h *SolutionHandler) Delete(c *gin.Context) {
	solutionID := c.Param("id")
	uid, err := uuid.Parse(solutionID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid solution id"})
		return
	}

	if err := h.service.Delete(uid); err != nil {
		utils.DeleteConflictResponse(c, err, "solution")
		return
	}

	c.JSON(200, gin.H{"message": "Solution deleted successfully"})
}
