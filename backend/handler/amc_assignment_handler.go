package handler

import (
	"log"
	"net/http"
	"strings"
	"time"

	"rbac/models"
	"rbac/repository"
	"rbac/service"
	"rbac/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AMCAssignmentHandler struct {
	service             *service.AMCAssignmentService
	uploader            utils.ImageUploader
	supportEngineerRepo *repository.SupportEngineerRepository
}

func NewAMCAssignmentHandler(
	service *service.AMCAssignmentService,
	uploader utils.ImageUploader,
	supportEngineerRepo *repository.SupportEngineerRepository,
) *AMCAssignmentHandler {
	return &AMCAssignmentHandler{
		service:             service,
		uploader:            uploader,
		supportEngineerRepo: supportEngineerRepo,
	}
}

/*
	=========================
	  ADMIN: ASSIGN AMC TO ENGINEER

=========================
*/
func (h *AMCAssignmentHandler) AssignAMC(c *gin.Context) {
	var req struct {
		CustomerSolutionID uuid.UUID `json:"customer_solution_id" binding:"required"`
		SupportEngineerID  uuid.UUID `json:"support_engineer_id" binding:"required"`
		AMCStartDate       time.Time `json:"amc_start_date" binding:"required"`
		AMCEndDate         time.Time `json:"amc_end_date" binding:"required"`
		Notes              string    `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	adminID := c.MustGet("user_id").(uuid.UUID)

	assignment := &models.AMCAssignment{
		CustomerSolutionID: req.CustomerSolutionID,
		SupportEngineerID:  req.SupportEngineerID,
		AssignedBy:         adminID,
		AMCStartDate:       req.AMCStartDate,
		AMCEndDate:         req.AMCEndDate,
		Notes:              req.Notes,
		Status:             "active",
	}

	if err := h.service.AssignAMC(assignment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "AMC assigned successfully",
		"assignment": assignment,
	})
}

/*
	=========================
	  ADMIN: GET ALL AMC ASSIGNMENTS

=========================
*/
func (h *AMCAssignmentHandler) GetAllAMCs(c *gin.Context) {
	amcs, err := h.service.GetAllAMCs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch AMCs"})
		return
	}

	c.JSON(http.StatusOK, amcs)
}

/*
	=========================
	  ADMIN/ENGINEER: GET AMC DETAILS

=========================
*/
func (h *AMCAssignmentHandler) GetAMCAssignment(c *gin.Context) {
	assignmentID := c.Param("id")
	id, err := uuid.Parse(assignmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
		return
	}

	assignment, err := h.service.GetAMCAssignment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		return
	}

	c.JSON(http.StatusOK, assignment)
}

/*
	=========================
	  ENGINEER: GET MY AMC ASSIGNMENTS

=========================
*/
func (h *AMCAssignmentHandler) GetMyAMCs(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// Get support engineer ID from user ID
	engineer, err := h.supportEngineerRepo.GetByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Support engineer not found"})
		return
	}

	amcs, err := h.service.GetEngineerAMCs(engineer.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch AMCs"})
		return
	}

	c.JSON(http.StatusOK, amcs)
}

/*
	=========================
	  ENGINEER: COMPLETE VISIT

=========================
*/
func (h *AMCAssignmentHandler) CompleteVisit(c *gin.Context) {
	visitID := c.Param("visit_id")
	id, err := uuid.Parse(visitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid visit ID"})
		return
	}

	var req struct {
		VisitDate time.Time `json:"visit_date" binding:"required"`
		Notes     string    `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.CompleteVisit(id, req.VisitDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Visit completed successfully",
	})
}

/*
	=========================
	  ENGINEER: UPLOAD PROOF/IMAGES
	  (Now accepts image URL from client-side upload)
=========================
*/

type ProofRequest struct {
	ImageURL    string `json:"image_url" binding:"required"`
	Description string `json:"description"`
}

func (h *AMCAssignmentHandler) UploadProof(c *gin.Context) {
	visitID := c.Param("visit_id")
	id, err := uuid.Parse(visitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid visit ID"})
		return
	}

	engineerID := c.MustGet("user_id").(uuid.UUID)

	// Get JSON request body with image URL (from AWS S3 upload)
	var req ProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.ImageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_url is required"})
		return
	}

	// Validate it's from AWS S3, local storage, or any HTTPS URL
	if !strings.Contains(req.ImageURL, "s3") && !strings.Contains(req.ImageURL, "localhost") && !strings.HasPrefix(req.ImageURL, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image URL"})
		return
	}

	log.Printf(
		"[AMC_PROOF_UPLOAD_SUCCESS] visit_id=%s engineer_id=%s image_url=%s",
		visitID,
		engineerID,
		req.ImageURL,
	)

	// Create proof record with AWS S3 URL
	proof := &models.AMCVisitProof{
		AMCVisitID:  id,
		ImagePath:   req.ImageURL, // Stores S3 URL
		Description: req.Description,
		UploadedBy:  engineerID,
	}

	if err := h.service.AddVisitProof(proof); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Proof uploaded successfully",
		"proof":   proof,
	})
}

/*
	=========================
	  GET VISIT PROOFS

=========================
*/
func (h *AMCAssignmentHandler) GetVisitProofs(c *gin.Context) {
	visitID := c.Param("visit_id")
	id, err := uuid.Parse(visitID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid visit ID"})
		return
	}

	proofs, err := h.service.GetVisitProofs(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch proofs"})
		return
	}

	c.JSON(http.StatusOK, proofs)
}

/*
	=========================
	  ADMIN: UPDATE AMC ASSIGNMENT

=========================
*/
func (h *AMCAssignmentHandler) UpdateAMCAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment id"})
		return
	}

	var req struct {
		SupportEngineerID *uuid.UUID `json:"support_engineer_id"`
		AMCStartDate      *time.Time `json:"amc_start_date"`
		AMCEndDate        *time.Time `json:"amc_end_date"`
		Status            *string    `json:"status"`
		Notes             *string    `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	err = h.service.UpdateAMCAssignment(id, &service.UpdateAMCAssignmentRequest{
		SupportEngineerID: req.SupportEngineerID,
		AMCStartDate:      req.AMCStartDate,
		AMCEndDate:        req.AMCEndDate,
		Status:            req.Status,
		Notes:             req.Notes,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.service.GetAMCAssignment(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "AMC assignment updated successfully"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "AMC assignment updated successfully",
		"assignment": updated,
	})
}

/*
	=========================
	  ADMIN: DELETE AMC ASSIGNMENT

=========================
*/
func (h *AMCAssignmentHandler) DeleteAMCAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment id"})
		return
	}

	if err := h.service.DeleteAMCAssignment(id); err != nil {
		utils.DeleteConflictResponse(c, err, "AMC assignment")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AMC assignment deleted successfully"})
}

/*
	=========================
	  DOWNLOAD PROOF IMAGE

=========================
*/
func (h *AMCAssignmentHandler) DownloadProof(c *gin.Context) {
	proofID := c.Param("proof_id")

	// In a real implementation, you'd fetch the proof from DB
	// For now, just validate the request
	if proofID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Proof ID required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Download proof endpoint",
	})
}
