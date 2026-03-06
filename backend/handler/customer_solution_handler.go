package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/models"
	"rbac/service"
)

type CustomerSolutionHandler struct {
	service *service.CustomerSolutionService
}

func NewCustomerSolutionHandler(
	s *service.CustomerSolutionService,
) *CustomerSolutionHandler {
	return &CustomerSolutionHandler{service: s}
}

/* =========================
   ADMIN: ASSIGN SOLUTION
========================= */

func (h *CustomerSolutionHandler) AssignSolution(c *gin.Context) {

	adminID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		CustomerID uuid.UUID `json:"customer_id" binding:"required"`
		SolutionID uuid.UUID `json:"solution_id" binding:"required"`
		PONumber   string    `json:"po_number" binding:"required"`

		ContractType models.ContractType `json:"contract_type" binding:"required,oneof=AMC Warranty Others/Chargeable"`

		Description string `json:"description"`
		// AMC
		AMCType      *models.AMCType `json:"amc_type"`
		AMCStartDate *time.Time      `json:"amc_start_date"`
		AMCEndDate   *time.Time      `json:"amc_end_date"`

		// Warranty
		WarrantyStartDate *time.Time `json:"warranty_start_date"`
		WarrantyEndDate   *time.Time `json:"warranty_end_date"`

		// Chargeable/Others
		ChargeableType *models.ChargeableType `json:"chargeable_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	err := h.service.AssignSolution(&service.AssignSolutionRequest{
		CustomerID: req.CustomerID,
		SolutionID: req.SolutionID,
		PONumber:   req.PONumber,

		ContractType: req.ContractType,
		Description:  req.Description,

		AMCType:      req.AMCType,
		AMCStartDate: req.AMCStartDate,
		AMCEndDate:   req.AMCEndDate,

		WarrantyStartDate: req.WarrantyStartDate,
		WarrantyEndDate:   req.WarrantyEndDate,

		ChargeableType: req.ChargeableType,

		AssignedBy: adminID,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to assign solution"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "solution assigned successfully",
	})
}

/* =========================
   GET CUSTOMER SOLUTIONS
========================= */

func (h *CustomerSolutionHandler) GetByCustomer(c *gin.Context) {

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	data, err := h.service.GetCustomerSolutions(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve solutions"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *CustomerSolutionHandler) Assign(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)

	var req service.AssignSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.AssignedBy = adminID

	if err := h.service.AssignSolution(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(201, gin.H{"message": "solution assigned"})
}

func (h *CustomerSolutionHandler) GetCustomerSolutions(c *gin.Context) {
	customerID := uuid.MustParse(c.Param("id"))
	list, _ := h.service.GetCustomerSolutions(customerID)
	c.JSON(200, list)
}

func (h *CustomerSolutionHandler) GetMySolutions(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	solutions, err := h.service.GetCustomerSolutionsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to retrieve solutions",
		})
		return
	}

	c.JSON(http.StatusOK, solutions)
}

/*
	=========================
	  GET ALL CUSTOMER SOLUTIONS

=========================
*/
func (h *CustomerSolutionHandler) GetAllCustomerSolutions(c *gin.Context) {
	solutions, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve solutions",
		})
		return
	}

	c.JSON(http.StatusOK, solutions)
}
