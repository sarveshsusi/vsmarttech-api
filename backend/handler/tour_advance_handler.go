package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/domain"
	"rbac/middleware"
	"rbac/models"
	"rbac/service"
)

type TourAdvanceHandler struct {
	service *service.TourAdvanceService
}

func NewTourAdvanceHandler(service *service.TourAdvanceService) *TourAdvanceHandler {
	return &TourAdvanceHandler{service: service}
}

type createTourAdvanceBody struct {
	CustomerID           uuid.UUID           `json:"customer_id" binding:"required"`
	WorkType             string              `json:"work_type" binding:"required"`
	PONumber             string              `json:"po_number"`
	TicketID             string              `json:"ticket_id"`
	PersonsTravelling    int                 `json:"persons_travelling"`
	Travellers           []tourTravellerBody `json:"travellers"`
	DaysPlanned          int                 `json:"days_planned"`
	TravelFrom           string              `json:"travel_from"`
	TravelTo             string              `json:"travel_to"`
	FoodExpense          string              `json:"food_expense" binding:"required"`
	LocalExpense         string              `json:"local_expense" binding:"required"`
	AccommodationExpense string              `json:"accommodation_expense" binding:"required"`
	TravelMode           string              `json:"travel_mode" binding:"required"`
	TravelExpense        string              `json:"travel_expense" binding:"required"`
}

type tourTravellerBody struct {
	EngineerID string `json:"engineer_id"`
	Name       string `json:"name"`
}

type approveTourAdvanceBody struct {
	ApprovedAmount  string `json:"approved_amount" binding:"required"`
	ApprovalRemarks string `json:"approval_remarks"`
}

type rejectTourAdvanceBody struct {
	RejectionRemarks string `json:"rejection_remarks" binding:"required"`
}

func (h *TourAdvanceHandler) Create(c *gin.Context) {
	var body createTourAdvanceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	actor, role := tourActor(c)
	view, err := h.service.Create(actor, role, service.CreateTourAdvanceInput{
		CustomerID:           body.CustomerID,
		WorkType:             body.WorkType,
		PONumber:             body.PONumber,
		TicketID:             body.TicketID,
		PersonsTravelling:    body.PersonsTravelling,
		Travellers:           tourTravellerInputs(body.Travellers),
		DaysPlanned:          body.DaysPlanned,
		TravelFrom:           body.TravelFrom,
		TravelTo:             body.TravelTo,
		FoodExpense:          body.FoodExpense,
		LocalExpense:         body.LocalExpense,
		AccommodationExpense: body.AccommodationExpense,
		TravelMode:           body.TravelMode,
		TravelExpense:        body.TravelExpense,
	})
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *TourAdvanceHandler) List(c *gin.Context) {
	actor, role := tourActor(c)
	list, err := h.service.List(actor, role, c.Query("status"))
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *TourAdvanceHandler) Get(c *gin.Context) {
	id, ok := tourID(c)
	if !ok {
		return
	}
	actor, role := tourActor(c)
	view, err := h.service.Get(actor, role, id)
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *TourAdvanceHandler) Approve(c *gin.Context) {
	id, ok := tourID(c)
	if !ok {
		return
	}
	var body approveTourAdvanceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	actor, role := tourActor(c)
	view, err := h.service.Approve(actor, role, id, body.ApprovedAmount, body.ApprovalRemarks)
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *TourAdvanceHandler) Reject(c *gin.Context) {
	id, ok := tourID(c)
	if !ok {
		return
	}
	var body rejectTourAdvanceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rejection reason is required"})
		return
	}
	actor, role := tourActor(c)
	view, err := h.service.Reject(actor, role, id, body.RejectionRemarks)
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *TourAdvanceHandler) Delete(c *gin.Context) {
	id, ok := tourID(c)
	if !ok {
		return
	}
	actor, role := tourActor(c)
	if err := h.service.Delete(actor, role, id); err != nil {
		writeTourError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *TourAdvanceHandler) BillSubmitted(c *gin.Context) {
	id, ok := tourID(c)
	if !ok {
		return
	}
	actor, role := tourActor(c)
	view, err := h.service.MarkBillSubmitted(actor, role, id)
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *TourAdvanceHandler) Process(c *gin.Context) {
	id, ok := tourID(c)
	if !ok {
		return
	}
	actor, role := tourActor(c)
	view, err := h.service.Process(actor, role, id)
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *TourAdvanceHandler) LookupCompanies(c *gin.Context) {
	rows, err := h.service.ListCompanies()
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *TourAdvanceHandler) LookupSites(c *gin.Context) {
	companyID, err := uuid.Parse(strings.TrimSpace(c.Query("company_id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer is required"})
		return
	}
	rows, err := h.service.ListSites(companyID)
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *TourAdvanceHandler) LookupTickets(c *gin.Context) {
	customerID, err := uuid.Parse(strings.TrimSpace(c.Query("customer_id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer is required"})
		return
	}
	rows, err := h.service.ListTickets(customerID)
	if err != nil {
		writeTourError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

func tourTravellerInputs(rows []tourTravellerBody) []service.TourTravellerInput {
	out := make([]service.TourTravellerInput, 0, len(rows))
	for _, row := range rows {
		out = append(out, service.TourTravellerInput{
			EngineerID: row.EngineerID,
			Name:       row.Name,
		})
	}
	return out
}

func tourActor(c *gin.Context) (uuid.UUID, models.Role) {
	actor := c.MustGet("user_id").(uuid.UUID)
	role, _ := c.Get(middleware.CtxUserRole)
	userRole, _ := role.(models.Role)
	return actor, userRole
}

func tourID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return uuid.Nil, false
	}
	return id, true
}

func writeTourError(c *gin.Context, err error) {
	var rule *domain.RuleError
	switch {
	case errors.As(err, &rule):
		c.JSON(http.StatusBadRequest, gin.H{"error": rule.Error()})
	case errors.Is(err, service.ErrTourNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
	case errors.Is(err, service.ErrInvalidTransition):
		c.JSON(http.StatusConflict, gin.H{"error": "invalid state transition"})
	case errors.Is(err, service.ErrTourForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	default:
		log.Printf("tour advance: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
	}
}
