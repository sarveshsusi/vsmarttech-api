package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/service"
)

type CustomerHandler struct {
	service *service.CustomerService
}

func NewCustomerHandler(s *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: s}
}

type CreateCustomerRequest struct {
	Name          string    `json:"name" binding:"required"`
	Email         string    `json:"email" binding:"required,email"`
	Password      string    `json:"password" binding:"required,min=8"`
	CompanyID     uuid.UUID `json:"company_id" binding:"required"`
	CompanyName   string    `json:"company_name" binding:"required"`
	Location      string    `json:"location" binding:"required"`
	Plant         string    `json:"plant" binding:"required"`
	Phone         string    `json:"phone"`
	Address       string    `json:"address"`
	ContactPerson string    `json:"contact_person" binding:"required"`
}

func (h *CustomerHandler) Create(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.CreateCustomer(
		req.Name,
		req.Email,
		req.Password,
		req.CompanyID,
		req.CompanyName,
		req.ContactPerson,
		req.Location,
		req.Plant,
		req.Phone,
		req.Address,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "customer created"})
}

func (h *CustomerHandler) GetAll(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	customers, total, err := h.service.GetAllCustomers(page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": customers,
		"meta": gin.H{
			"page":  page,
			"limit": 3,
			"total": total,
		},
	})
}
