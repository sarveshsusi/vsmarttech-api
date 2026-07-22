package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"rbac/models"
	"rbac/repository"
	"rbac/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AssetHandler struct {
	service *service.AssetService
}

func NewAssetHandler(s *service.AssetService) *AssetHandler {
	return &AssetHandler{service: s}
}

func parseOptionalUUID(raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

type assetBody struct {
	CompanyID          string  `json:"company_id" binding:"required"`
	CustomerID         string  `json:"customer_id"`
	CustomerSolutionID string  `json:"customer_solution_id"`
	SerialNumber       string  `json:"serial_number" binding:"required"`
	Name               string  `json:"name" binding:"required"`
	Model              string  `json:"model"`
	Category           string  `json:"category"`
	SiteLocation       string  `json:"site_location"`
	Notes              string  `json:"notes"`
	Status             string  `json:"status"`
	InstalledAt        *string `json:"installed_at"`
}

func (h *AssetHandler) bindInput(body assetBody) (service.AssetInput, error) {
	companyID, err := uuid.Parse(body.CompanyID)
	if err != nil {
		return service.AssetInput{}, err
	}
	customerID, err := parseOptionalUUID(body.CustomerID)
	if err != nil {
		return service.AssetInput{}, err
	}
	csID, err := parseOptionalUUID(body.CustomerSolutionID)
	if err != nil {
		return service.AssetInput{}, err
	}

	var installedAt *time.Time
	if body.InstalledAt != nil && strings.TrimSpace(*body.InstalledAt) != "" {
		if t, e := time.Parse("2006-01-02", strings.TrimSpace(*body.InstalledAt)); e == nil {
			installedAt = &t
		}
	}

	return service.AssetInput{
		CompanyID:          companyID,
		CustomerID:         customerID,
		CustomerSolutionID: csID,
		SerialNumber:       body.SerialNumber,
		Name:               body.Name,
		Model:              body.Model,
		Category:           body.Category,
		SiteLocation:       body.SiteLocation,
		Notes:              body.Notes,
		Status:             models.AssetStatus(body.Status),
		InstalledAt:        installedAt,
	}, nil
}

func (h *AssetHandler) Create(c *gin.Context) {
	var body assetBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	in, err := h.bindInput(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ids"})
		return
	}
	adminID := c.MustGet("user_id").(uuid.UUID)
	asset, err := h.service.Create(adminID, in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, asset)
}

func (h *AssetHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	var body assetBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	in, err := h.bindInput(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ids"})
		return
	}
	asset, err := h.service.Update(id, in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, asset)
}

func (h *AssetHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	rows, total, err := h.service.List(repository.AssetListFilter{
		Search:             c.Query("search"),
		CompanyID:          c.Query("company_id"),
		CustomerSolutionID: c.Query("customer_solution_id"),
		Status:             c.Query("status"),
		Limit:              pageSize,
		Offset:             (page - 1) * pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load assets"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AssetHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	asset, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	c.JSON(http.StatusOK, asset)
}

func (h *AssetHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
