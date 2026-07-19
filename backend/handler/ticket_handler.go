package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/models"
	"rbac/service"
	"rbac/utils"
)

type TicketHandler struct {
	service  *service.TicketService
	uploader utils.ImageUploader
}

func NewTicketHandler(
	s *service.TicketService,
	uploader utils.ImageUploader,
) *TicketHandler {
	return &TicketHandler{
		service:  s,
		uploader: uploader,
	}
}

/* =========================
   CUSTOMER: CREATE TICKET
========================= */

type CustomerCreateTicketRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type AdminAssignTicketRequest struct {
	CustomerSolutionID uuid.UUID              `json:"customer_solution_id" binding:"required"`
	EngineerID         uuid.UUID              `json:"engineer_id" binding:"required"`
	Priority           models.TicketPriority  `json:"priority" binding:"required"`
	SupportMode        models.SupportMode     `json:"support_mode" binding:"required"`
	ServiceCallType    models.ServiceCallType `json:"service_call_type" binding:"required"`
}

func (h *TicketHandler) CreateTicket(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req CustomerCreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	log.Printf("[CREATE_TICKET] userID=%s title=%s", userID, req.Title)

	ticket, err := h.service.CustomerCreateTicket(
		userID,
		req.Title,
		req.Description,
	)
	if err != nil {
		log.Printf("[CREATE_TICKET_ERROR] userID=%s error=%v", userID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to create ticket"})
		return
	}

	log.Printf("[CREATE_TICKET_SUCCESS] ticketID=%s customerID=%s", ticket.ID, ticket.CustomerID)
	c.JSON(http.StatusCreated, ticket)
}

/* =========================
   ADMIN: CREATE TICKET
========================= */

type AdminCreateTicketRequest struct {
	CustomerID         uuid.UUID `json:"customer_id" binding:"required"`
	CustomerSolutionID uuid.UUID `json:"customer_solution_id" binding:"required"`

	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`

	EngineerID  uuid.UUID             `json:"engineer_id" binding:"required"`
	Priority    models.TicketPriority `json:"priority" binding:"required"`
	SupportMode models.SupportMode    `json:"support_mode" binding:"required"`
}

func (h *TicketHandler) AdminCreateTicket(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)

	var req AdminCreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ticket, err := h.service.AdminCreateTicketAndAssign(
		req.CustomerID,
		req.CustomerSolutionID,
		req.Title,
		req.Description,
		req.EngineerID,
		req.Priority,
		req.SupportMode,
		adminID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to create ticket"})
		return
	}

	c.JSON(http.StatusCreated, ticket)
}

/*
	=========================
	  ADMIN: ASSIGN TICKET

=========================
*/
type AssignTicketRequest struct {
	TicketID           string                 `json:"ticket_id" binding:"required"`
	CustomerSolutionID uuid.UUID              `json:"customer_solution_id" binding:"required"`
	EngineerID         uuid.UUID              `json:"engineer_id" binding:"required"`
	Priority           models.TicketPriority  `json:"priority" binding:"required"`
	SupportMode        models.SupportMode     `json:"support_mode" binding:"required"`
	ServiceCallType    models.ServiceCallType `json:"service_call_type" binding:"required"`
}

func (h *TicketHandler) AssignTicket(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)

	var req AssignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.AdminAssignTicket(
		req.TicketID,
		req.CustomerSolutionID,
		req.EngineerID,
		adminID,
		req.Priority,
		req.SupportMode,
		req.ServiceCallType,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned"})
}

/* =========================
   SUPPORT: START TICKET
========================= */

func (h *TicketHandler) StartTicket(c *gin.Context) {
	engineerID := c.MustGet("user_id").(uuid.UUID)

	// Extract ticket ID from URL path (e.g., /tickets/VS/06/26/1/start)
	// Trim the leading slash from catch-all parameter
	ticketID := strings.TrimPrefix(c.Param("id"), "/")
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id is required"})
		return
	}

	if err := h.service.StartTicket(ticketID, engineerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket in progress"})
}

/* =========================
   SUPPORT: CLOSE TICKET
========================= */

/* =========================
   ENGINEER: CLOSE TICKET
   (Now accepts image URL from client-side upload)
========================= */

type CloseTicketRequest struct {
	TicketID       string `json:"ticket_id" binding:"required"`
	ProofURL       string `json:"proof_url" binding:"required"`
	SupportComment string `json:"support_comment" binding:"required"`
}

func (h *TicketHandler) CloseTicket(c *gin.Context) {
	engineerID := c.MustGet("user_id").(uuid.UUID)

	// Accept JSON request with proof URL (from AWS S3 upload)
	var req CloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.ProofURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proof_url is required"})
		return
	}

	if req.SupportComment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "support_comment is required"})
		return
	}

	// Validate URL is from AWS S3, local storage, or any HTTPS URL
	if !strings.Contains(req.ProofURL, "s3") && !strings.Contains(req.ProofURL, "localhost") && !strings.Contains(req.ProofURL, "/uploads/") && !strings.HasPrefix(req.ProofURL, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proof URL"})
		return
	}

	log.Printf(
		"[TICKET_CLOSE] ticket_id=%s engineer_id=%s proof_url=%s",
		req.TicketID,
		engineerID,
		req.ProofURL,
	)

	if err := h.service.CloseTicket(req.TicketID, engineerID, req.ProofURL, req.SupportComment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket closed", "proof": req.ProofURL})
}

/* =========================
   ADMIN: CLOSE TICKET (On behalf of support engineer)
   Closes ticket without requiring proof image
========================= */

type AdminCloseTicketRequest struct {
	TicketID     string `json:"ticket_id" binding:"required"`
	AdminComment string `json:"admin_comment" binding:"required"`
}

func (h *TicketHandler) AdminCloseTicket(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)

	var req AdminCloseTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.AdminComment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin_comment is required"})
		return
	}

	log.Printf(
		"[ADMIN_CLOSE_TICKET] ticket_id=%s admin_id=%s",
		req.TicketID,
		adminID,
	)

	if err := h.service.AdminCloseTicket(req.TicketID, adminID, req.AdminComment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket closed by admin"})
}

/* =========================
   ADMIN: GET ALL TICKETS
========================= */

func (h *TicketHandler) GetAdminTickets(c *gin.Context) {
	tickets, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid request"})
		return
	}
	c.JSON(http.StatusOK, tickets)
}

type AdminCreateTicketAndAssignRequest struct {
	CustomerID         uuid.UUID             `json:"customer_id" binding:"required"`
	CustomerSolutionID uuid.UUID             `json:"customer_solution_id" binding:"required"`
	Title              string                `json:"title" binding:"required"`
	Description        string                `json:"description" binding:"required"`
	EngineerID         uuid.UUID             `json:"engineer_id" binding:"required"`
	Priority           models.TicketPriority `json:"priority" binding:"required"`
	SupportMode        models.SupportMode    `json:"support_mode" binding:"required"`
}

func (h *TicketHandler) AdminCreateTicketAndAssign(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)

	var req AdminCreateTicketAndAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ticket, err := h.service.AdminCreateTicketAndAssign(
		req.CustomerID,
		req.CustomerSolutionID,
		req.Title,
		req.Description,
		req.EngineerID,
		req.Priority,
		req.SupportMode,
		adminID,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusCreated, ticket)
}

/*
	=========================
	  CUSTOMER: GET TICKET BY ID

=========================
*/
func (h *TicketHandler) GetTicketById(c *gin.Context) {
	ticketID := c.Query("id") // Read from query parameter
	if ticketID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_id is required"})
		return
	}
	
	userID := c.MustGet("user_id").(uuid.UUID)

	// Get ticket from service
	ticket, err := h.service.GetTicketById(ticketID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

/* =========================
   ADMIN: REASSIGN TICKET
========================= */

type ReassignTicketRequest struct {
	TicketID   string    `json:"ticket_id" binding:"required"`
	EngineerID uuid.UUID `json:"engineer_id" binding:"required"`
}

func (h *TicketHandler) ReassignTicket(c *gin.Context) {
	adminID := c.MustGet("user_id").(uuid.UUID)

	var req ReassignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ticket, err := h.service.ReassignTicket(req.TicketID, req.EngineerID, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

/* =========================
   DEPRECATED: IMAGEKIT AUTH (No longer used)
========================= */

// GetImageKitAuthToken is deprecated - use POST /api/v1/upload/proof instead
func (h *TicketHandler) GetImageKitAuthToken(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"error":   "ImageKit authentication is deprecated",
		"message": "Use POST /api/v1/upload/proof instead for AWS S3 uploads",
	})
}

/* =========================
   UPLOAD: S3 FILE UPLOAD
========================= */

func (h *TicketHandler) UploadProofImage(c *gin.Context) {
	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Validate file type (only images)
	allowedTypes := []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	isAllowed := false
	for _, allowedType := range allowedTypes {
		if file.Header.Get("Content-Type") == allowedType {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPEG, PNG, WebP, and GIF allowed"})
		return
	}

	// Upload to S3 (with compression if needed)
	imageURL, err := h.uploader.Upload(file)
	if err != nil {
		log.Printf("[UPLOAD_ERROR] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload image"})
		return
	}

	log.Printf("[UPLOAD_SUCCESS] Image uploaded to: %s", imageURL)
	c.JSON(http.StatusOK, gin.H{
		"url":     imageURL,
		"message": "Image uploaded successfully",
	})
}
