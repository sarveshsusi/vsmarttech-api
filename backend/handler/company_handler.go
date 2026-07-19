package handler

import (
	"net/http"

	"rbac/dto"
	"rbac/service"
	"rbac/utils"

	"github.com/gin-gonic/gin"
)

type CompanyHandler struct {
	service service.CompanyService
}

func NewCompanyHandler(service service.CompanyService) *CompanyHandler {
	return &CompanyHandler{service}
}

// POST /admin/companies
func (h *CompanyHandler) CreateCompany(c *gin.Context) {
	var req dto.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	company, err := h.service.CreateCompany(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":   company.ID,
		"name": company.Name,
	})
}

// GET /admin/companies
func (h *CompanyHandler) GetCompanies(c *gin.Context) {
	companies, err := h.service.GetCompanies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch companies"})
		return
	}

	resp := make([]dto.CompanyResponse, 0, len(companies))
	for _, cpy := range companies {
		resp = append(resp, dto.CompanyResponse{
			ID:   cpy.ID.String(),
			Name: cpy.Name,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// PUT /admin/companies/:id
func (h *CompanyHandler) EditCompany(c *gin.Context) {
	id := c.Param("id")

	var req dto.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	company, err := h.service.UpdateCompany(id, req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   company.ID,
		"name": company.Name,
	})
}

// DELETE /admin/companies/:id
func (h *CompanyHandler) DeleteCompany(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteCompany(id); err != nil {
		utils.DeleteConflictResponse(c, err, "company")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Company deleted successfully"})
}
