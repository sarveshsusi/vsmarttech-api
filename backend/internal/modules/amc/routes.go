package amc

import (
	"github.com/gin-gonic/gin"

	"rbac/handler"
)

// Handlers groups AMC / contract HTTP handlers.
type Handlers struct {
	Contract *handler.ContractHandler
	AMC      *handler.AMCAssignmentHandler
}

// RegisterAdmin mounts AMC scheduler and contract admin routes.
func RegisterAdmin(admin *gin.RouterGroup, h Handlers) {
	admin.GET("/contracts/amc", h.Contract.GetAMCContracts)
	admin.GET("/contracts/warranty", h.Contract.GetWarrantyContracts)
	admin.POST("/contracts/check-expiry", h.Contract.TriggerExpiryCheck)

	admin.POST("/amc-assignments", h.AMC.AssignAMC)
	admin.GET("/amc-assignments", h.AMC.GetAllAMCs)
	admin.GET("/amc-assignments/:id", h.AMC.GetAMCAssignment)
	admin.PUT("/amc-assignments/:id", h.AMC.UpdateAMCAssignment)
	admin.DELETE("/amc-assignments/:id", h.AMC.DeleteAMCAssignment)
	admin.GET("/amc-assignments/:id/proofs", h.AMC.GetVisitProofs)
	admin.PUT("/amc-visits/:visit_id/reschedule", h.AMC.RescheduleVisit)
}

// RegisterSupport mounts engineer AMC visit routes.
func RegisterSupport(support *gin.RouterGroup, h Handlers) {
	support.GET("/amc-assignments", h.AMC.GetMyAMCs)
	support.GET("/amc-assignments/:id", h.AMC.GetAMCAssignment)
	support.PUT("/amc-visits/:visit_id/complete", h.AMC.CompleteVisit)
	support.POST("/amc-visits/:visit_id/proofs", h.AMC.UploadProof)
	support.GET("/amc-visits/:visit_id/proofs", h.AMC.GetVisitProofs)
}
