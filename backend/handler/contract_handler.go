package handler

import (
	"net/http"

	"rbac/service"

	"github.com/gin-gonic/gin"
)

type ContractHandler struct {
	contractService *service.ContractExpiryService
}

func NewContractHandler(contractService *service.ContractExpiryService) *ContractHandler {
	return &ContractHandler{
		contractService: contractService,
	}
}

/* =========================
   GET ALL AMC CONTRACTS
========================= */

// GetAMCContracts returns all AMC contracts with customer details
// @Summary Get all AMC contracts
// @Tags Contracts
// @Produce json
// @Success 200 {array} service.ContractWithCustomer
// @Router /admin/contracts/amc [get]
func (h *ContractHandler) GetAMCContracts(c *gin.Context) {
	contracts, err := h.contractService.GetAllAMCContractsWithDetails()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch AMC contracts"})
		return
	}

	c.JSON(http.StatusOK, contracts)
}

/* =========================
   GET ALL WARRANTY CONTRACTS
========================= */

// GetWarrantyContracts returns all Warranty contracts with customer details
// @Summary Get all Warranty contracts
// @Tags Contracts
// @Produce json
// @Success 200 {array} service.ContractWithCustomer
// @Router /admin/contracts/warranty [get]
func (h *ContractHandler) GetWarrantyContracts(c *gin.Context) {
	contracts, err := h.contractService.GetAllWarrantyContractsWithDetails()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch warranty contracts"})
		return
	}

	c.JSON(http.StatusOK, contracts)
}

/* =========================
   TRIGGER MANUAL CHECK (ADMIN ONLY)
========================= */

// TriggerExpiryCheck manually triggers the contract expiry check
// @Summary Manually trigger expiry check
// @Tags Contracts
// @Produce json
// @Success 200 {object} map[string]string
// @Router /admin/contracts/check-expiry [post]
func (h *ContractHandler) TriggerExpiryCheck(c *gin.Context) {
	go h.contractService.CheckAndNotifyExpiringContracts()
	c.JSON(http.StatusOK, gin.H{"message": "Contract expiry check triggered"})
}
