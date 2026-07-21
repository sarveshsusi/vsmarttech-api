package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/domain"
	"rbac/models"
	"rbac/repository"
)

type TicketService struct {
	db                   *gorm.DB
	ticketRepo           *repository.TicketRepository
	customerRepo         *repository.CustomerRepository
	customerSolutionRepo *repository.CustomerSolutionRepository
	notificationService  *NotificationService
	escalationRepo       *repository.TicketEscalationRepository
	visitRepo            *repository.ServiceVisitRepository
}

func assertTicketTransition(from, to models.TicketStatus) error {
	if !domain.CanTransition(from, to) {
		return fmt.Errorf("invalid status transition from %s to %s", from, to)
	}
	return nil
}

func statusPtr(s models.TicketStatus) *string {
	v := string(s)
	return &v
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func (s *TicketService) logTicketEvent(event *models.TicketEvent) {
	if event == nil {
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if err := s.ticketRepo.CreateEvent(event); err != nil {
		log.Printf("[TICKET_EVENT_ERROR] ticket=%s type=%s err=%v", event.TicketID, event.EventType, err)
	}
}

func (s *TicketService) logTicketEventTx(tx *gorm.DB, event *models.TicketEvent) error {
	if event == nil {
		return nil
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	return repository.NewTicketRepository(tx).CreateEventTx(tx, event)
}

func NewTicketService(
	db *gorm.DB,
	ticketRepo *repository.TicketRepository,
	customerRepo *repository.CustomerRepository,
	customerSolutionRepo *repository.CustomerSolutionRepository,
	notificationService *NotificationService,
) *TicketService {
	return &TicketService{
		db:                   db,
		ticketRepo:           ticketRepo,
		customerRepo:         customerRepo,
		customerSolutionRepo: customerSolutionRepo,
		notificationService:  notificationService,
		escalationRepo:       repository.NewTicketEscalationRepository(db),
		visitRepo:            repository.NewServiceVisitRepository(db),
	}
}

//
// =========================
// ADMIN / CUSTOMER: CREATE TICKET (PO BASED)
// =========================
//
// func (s *TicketService) createTicket(
// 	customerID uuid.UUID,
// 	title string,
// 	description string,
// 	createdBy uuid.UUID,
// ) (*models.Ticket, error) {

// 	cs, err := s.customerSolutionRepo.GetByCustomerAndPO(customerID, poNumber)
// 	if err != nil {
// 		return nil, errors.New("invalid PO for customer")
// 	}

// 	if s.customerSolutionRepo.IsPOExpired(cs) {
// 		return nil, errors.New("contract expired")
// 	}

// 	now := time.Now()

// 	ticket := &models.Ticket{
// 		ID:                 uuid.New(),
// 		CustomerID:         customerID,
// 		CustomerSolutionID: cs.ID,

// 		// immutable snapshots
// 		SolutionTitle: cs.Solution.Title,
// 		PONumber:      cs.PONumber,
// 		ContractType:  string(cs.ContractType),
// ServiceCallType: models.ServiceCallType(cs.ContractType),
// 		Title:       title,
// 		Description: description,
// 		Status:      models.StatusOpen,

// 		CreatedBy: createdBy,
// 		CreatedAt: now,
// 		UpdatedAt: now,
// 	}

// 	if err := s.ticketRepo.Create(ticket); err != nil {
// 		return nil, err
// 	}

// 	return ticket, nil
// }

// func (s *TicketService) AdminCreateTicket(
// 	customerID uuid.UUID,
// 	poNumber string,
// 	title string,
// 	description string,
// 	adminID uuid.UUID,
// ) (*models.Ticket, error) {

// 	return s.createTicket(customerID, poNumber, title, description, adminID)
// }

func (s *TicketService) CustomerCreateTicket(
	userID uuid.UUID,
	title string,
	description string,
) (*models.Ticket, error) {

	customer, err := s.customerRepo.GetByUserID(userID)
	if err != nil {
		log.Printf("[CUSTOMER_CREATE_TICKET] customer lookup failed for userID=%s error=%v", userID, err)
		return nil, errors.New("customer profile not found")
	}

	log.Printf("[CUSTOMER_CREATE_TICKET] customerID=%s found for userID=%s", customer.ID, userID)

	now := time.Now()

	// Generate custom ticket ID
	ticketID, err := s.ticketRepo.GenerateNextTicketID()
	if err != nil {
		log.Printf("[CUSTOMER_CREATE_TICKET_ERROR] Failed to generate ticket ID: %v", err)
		return nil, errors.New("failed to generate ticket ID")
	}

	ticket := &models.Ticket{
		ID:          ticketID,
		CustomerID:  customer.ID,
		Title:       title,
		Description: description,
		Status:      models.StatusOpen,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	log.Printf("[CUSTOMER_CREATE_TICKET] inserting ticket with customerID=%s", customer.ID)

	if err := s.ticketRepo.Create(ticket); err != nil {
		log.Printf("[CUSTOMER_CREATE_TICKET_ERROR] insert failed customerID=%s error=%v", customer.ID, err)
		return nil, err
	}

	s.logTicketEvent(&models.TicketEvent{
		TicketID:    ticket.ID,
		EventType:   models.TicketEventCreated,
		ActorUserID: userID,
		ToStatus:    statusPtr(models.StatusOpen),
		CreatedAt:   now,
	})

	log.Printf("[CUSTOMER_CREATE_TICKET_SUCCESS] ticketID=%s customerID=%s", ticket.ID, customer.ID)

	// Send notification for ticket creation
	if s.notificationService != nil {
		go s.notificationService.NotifyTicketCreated(ticket.ID, customer.ID)
	}

	return ticket, nil
}

// =========================
// ADMIN: ASSIGN TICKET
// =========================
func (s *TicketService) AssignTicket(
	ticketID string,
	engineerID uuid.UUID,
	adminID uuid.UUID,
	priority models.TicketPriority,
	supportMode models.SupportMode,
	serviceType models.ServiceCallType,
) error {

	return s.db.Transaction(func(tx *gorm.DB) error {

		repo := repository.NewTicketRepository(tx)
		now := time.Now()

		ticket, err := repo.GetByID(ticketID)
		if err != nil {
			return err
		}

		if err := assertTicketTransition(ticket.Status, models.StatusAssigned); err != nil {
			return err
		}

		ok, err := repo.SupportEngineerExists(engineerID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("invalid support engineer")
		}

		assigned, err := repo.IsAssigned(ticketID)
		if err != nil {
			return err
		}
		if assigned {
			return errors.New("ticket already assigned")
		}

		if err := repo.AssignEngineer(&models.TicketAssignment{
			TicketID:   ticketID,
			EngineerID: engineerID,
			AssignedBy: adminID,
			AssignedAt: now,
		}); err != nil {
			return err
		}

		if err := repo.UpdateFields(ticketID, map[string]interface{}{
			"engineer_id":       engineerID,
			"priority":          priority,
			"support_mode":      supportMode,
			"service_call_type": serviceType,
			"status":            models.StatusAssigned,
			"updated_at":        now,
		}); err != nil {
			return err
		}

		if err := repo.CreateStatusHistory(&models.TicketStatusHistory{
			TicketID:  ticketID,
			OldStatus: string(models.StatusOpen),
			NewStatus: string(models.StatusAssigned),
			ChangedBy: adminID,
			ChangedAt: now,
		}); err != nil {
			return err
		}

		return s.logTicketEventTx(tx, &models.TicketEvent{
			TicketID:     ticketID,
			EventType:    models.TicketEventAssigned,
			ActorUserID:  adminID,
			FromStatus:   statusPtr(models.StatusOpen),
			ToStatus:     statusPtr(models.StatusAssigned),
			ToEngineerID: uuidPtr(engineerID),
			CreatedAt:    now,
		})
	})
}

// =========================
// SUPPORT: START WORK
// =========================
func (s *TicketService) StartTicket(
	ticketID string,
	userID uuid.UUID,
) error {
	log.Printf("[START_TICKET] Starting for ticketID=%s userID=%s", ticketID, userID)

	// ✅ GET ENGINEER ID FROM USER ID
	var engineer models.SupportEngineer
	if err := s.db.Where("user_id = ?", userID).First(&engineer).Error; err != nil {
		log.Printf("[START_TICKET_ERROR] Support engineer not found for user_id=%s: %v", userID, err)
		return errors.New("support engineer profile not found")
	}

	log.Printf("[START_TICKET] Found engineer: id=%s user_id=%s", engineer.ID, engineer.UserID)

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		log.Printf("[START_TICKET_ERROR] Failed to get ticket: %v", err)
		return err
	}

	log.Printf("[START_TICKET] Current status: %s, assigned to engineer: %v", ticket.Status, ticket.EngineerID)

	// ✅ VERIFY ENGINEER IS ASSIGNED TO THIS TICKET
	if ticket.EngineerID == nil || *ticket.EngineerID != engineer.ID {
		log.Printf("[START_TICKET_ERROR] Engineer %s is not assigned to this ticket (ticket assigned to %v)", engineer.ID, ticket.EngineerID)
		return errors.New("you are not assigned to this ticket")
	}

	if err := assertTicketTransition(ticket.Status, models.StatusInProgress); err != nil {
		log.Printf("[START_TICKET_ERROR] %v", err)
		return err
	}

	now := time.Now()

	log.Printf("[START_TICKET] Updating status to In Progress")
	if err := s.ticketRepo.UpdateFields(ticketID, map[string]interface{}{
		"status":     models.StatusInProgress,
		"updated_at": now,
	}); err != nil {
		log.Printf("[START_TICKET_ERROR] Failed to update status: %v", err)
		return err
	}

	log.Printf("[START_TICKET] Creating status history")
	if err := s.ticketRepo.CreateStatusHistory(&models.TicketStatusHistory{
		TicketID:  ticketID,
		OldStatus: string(models.StatusAssigned),
		NewStatus: string(models.StatusInProgress),
		ChangedBy: engineer.ID,
		ChangedAt: now,
	}); err != nil {
		log.Printf("[START_TICKET_ERROR] Failed to create status history: %v", err)
		return err
	}

	s.logTicketEvent(&models.TicketEvent{
		TicketID:       ticketID,
		EventType:      models.TicketEventStarted,
		ActorUserID:    userID,
		FromStatus:     statusPtr(models.StatusAssigned),
		ToStatus:       statusPtr(models.StatusInProgress),
		FromEngineerID: uuidPtr(engineer.ID),
		ToEngineerID:   uuidPtr(engineer.ID),
		CreatedAt:      now,
	})

	// Send notification for status change
	if s.notificationService != nil {
		log.Printf("[START_TICKET] Sending status change notifications")
		go s.notificationService.NotifyTicketStatusChanged(
			ticketID,
			string(models.StatusAssigned),
			string(models.StatusInProgress),
			engineer.ID,
		)
	}

	log.Printf("[START_TICKET_SUCCESS] Ticket status changed to In Progress")
	return nil
}

// =========================
// SUPPORT: CLOSE TICKET
// =========================
func (s *TicketService) CloseTicket(
	ticketID string,
	userID uuid.UUID,
	proofImageURL string,
	supportComment string,
) error {
	log.Printf("[CLOSE_TICKET] Starting for ticketID=%s userID=%s", ticketID, userID)

	if proofImageURL == "" {
		log.Printf("[CLOSE_TICKET_ERROR] No closure proof image provided")
		return errors.New("closure proof image required")
	}

	if supportComment == "" {
		log.Printf("[CLOSE_TICKET_ERROR] No support comment provided")
		return errors.New("support comment is required")
	}

	// ✅ GET ENGINEER ID FROM USER ID
	var engineer models.SupportEngineer
	if err := s.db.Where("user_id = ?", userID).First(&engineer).Error; err != nil {
		log.Printf("[CLOSE_TICKET_ERROR] Support engineer not found for user_id=%s: %v", userID, err)
		return errors.New("support engineer profile not found")
	}

	log.Printf("[CLOSE_TICKET] Found engineer: id=%s user_id=%s", engineer.ID, engineer.UserID)

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		log.Printf("[CLOSE_TICKET_ERROR] Failed to get ticket: %v", err)
		return err
	}

	log.Printf("[CLOSE_TICKET] Current status: %s, assigned to engineer: %v", ticket.Status, ticket.EngineerID)

	// ✅ VERIFY ENGINEER IS ASSIGNED TO THIS TICKET
	if ticket.EngineerID == nil || *ticket.EngineerID != engineer.ID {
		log.Printf("[CLOSE_TICKET_ERROR] Engineer %s is not assigned to this ticket (ticket assigned to %v)", engineer.ID, ticket.EngineerID)
		return errors.New("you are not assigned to this ticket")
	}

	if err := assertTicketTransition(ticket.Status, models.StatusClosed); err != nil {
		log.Printf("[CLOSE_TICKET_ERROR] %v", err)
		return err
	}

	visitCount, err := s.visitRepo.CountByTicketID(ticketID)
	if err != nil {
		log.Printf("[CLOSE_TICKET_ERROR] Failed to count field visits: %v", err)
		return errors.New("failed to verify field visits")
	}
	if visitCount < 1 {
		log.Printf("[CLOSE_TICKET_ERROR] No field visits logged for ticket %s", ticketID)
		return errors.New("log at least one field visit before closing this ticket")
	}

	now := time.Now()

	log.Printf("[CLOSE_TICKET] Updating status to Closed with data: proof_image_url=%s, support_comment=%s", proofImageURL, supportComment)
	updateFields := map[string]interface{}{
		"status":              models.StatusClosed,
		"closure_proof_image": proofImageURL,
		"support_comment":     supportComment,
		"closed_at":           now,
		"updated_at":          now,
	}
	log.Printf("[CLOSE_TICKET] Fields to update: %+v", updateFields)

	if err := s.ticketRepo.UpdateFields(ticketID, updateFields); err != nil {
		log.Printf("[CLOSE_TICKET_ERROR] Failed to update ticket: %v", err)
		return err
	}

	log.Printf("[CLOSE_TICKET] Successfully updated ticket in database")

	log.Printf("[CLOSE_TICKET] Creating status history")
	if err := s.ticketRepo.CreateStatusHistory(&models.TicketStatusHistory{
		TicketID:  ticketID,
		OldStatus: string(models.StatusInProgress),
		NewStatus: string(models.StatusClosed),
		ChangedBy: engineer.ID,
		ChangedAt: now,
	}); err != nil {
		log.Printf("[CLOSE_TICKET_ERROR] Failed to create status history: %v", err)
		return err
	}

	s.logTicketEvent(&models.TicketEvent{
		TicketID:       ticketID,
		EventType:      models.TicketEventClosed,
		ActorUserID:    userID,
		FromStatus:     statusPtr(models.StatusInProgress),
		ToStatus:       statusPtr(models.StatusClosed),
		FromEngineerID: uuidPtr(engineer.ID),
		ToEngineerID:   uuidPtr(engineer.ID),
		Note:           supportComment,
		CreatedAt:      now,
	})

	// Send notification for ticket closure
	if s.notificationService != nil {
		log.Printf("[CLOSE_TICKET] Sending closure notifications")
		go s.notificationService.NotifyTicketClosed(ticketID, supportComment)
	}

	// Resolve any pending escalations for this ticket
	if err := s.escalationRepo.ResolveByTicket(ticketID); err != nil {
		log.Printf("[CLOSE_TICKET_WARN] Failed to resolve escalations for ticket %s: %v", ticketID, err)
		// Don't return error, ticket is already closed
	} else {
		log.Printf("[CLOSE_TICKET] Escalations resolved for ticket %s", ticketID)
	}

	log.Printf("[CLOSE_TICKET_SUCCESS] Ticket closed successfully")
	return nil
}

// =========================
// ADMIN: CLOSE TICKET (on behalf of support engineer)
// =========================
func (s *TicketService) AdminCloseTicket(
	ticketID string,
	adminID uuid.UUID,
	adminComment string,
) error {
	log.Printf("[ADMIN_CLOSE_TICKET] Starting for ticketID=%s by adminID=%s", ticketID, adminID)

	if adminComment == "" {
		log.Printf("[ADMIN_CLOSE_TICKET_ERROR] Admin comment is required")
		return errors.New("admin comment is required")
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		log.Printf("[ADMIN_CLOSE_TICKET_ERROR] Failed to get ticket: %v", err)
		return err
	}

	log.Printf("[ADMIN_CLOSE_TICKET] Current status: %s", ticket.Status)

	if err := assertTicketTransition(ticket.Status, models.StatusClosed); err != nil {
		log.Printf("[ADMIN_CLOSE_TICKET_ERROR] %v", err)
		return err
	}

	now := time.Now()

	log.Printf("[ADMIN_CLOSE_TICKET] Updating status to Closed with admin comment: %s", adminComment)
	updateFields := map[string]interface{}{
		"status":          models.StatusClosed,
		"support_comment": adminComment,
		"closed_at":       now,
		"updated_at":      now,
	}

	if err := s.ticketRepo.UpdateFields(ticketID, updateFields); err != nil {
		log.Printf("[ADMIN_CLOSE_TICKET_ERROR] Failed to update ticket: %v", err)
		return err
	}

	log.Printf("[ADMIN_CLOSE_TICKET] Successfully updated ticket in database")

	log.Printf("[ADMIN_CLOSE_TICKET] Creating status history")
	oldStatus := ticket.Status
	if err := s.ticketRepo.CreateStatusHistory(&models.TicketStatusHistory{
		TicketID:  ticketID,
		OldStatus: string(oldStatus),
		NewStatus: string(models.StatusClosed),
		ChangedBy: adminID,
		ChangedAt: now,
	}); err != nil {
		log.Printf("[ADMIN_CLOSE_TICKET_ERROR] Failed to create status history: %v", err)
		return err
	}

	var fromEng *uuid.UUID
	if ticket.EngineerID != nil {
		fromEng = ticket.EngineerID
	}
	s.logTicketEvent(&models.TicketEvent{
		TicketID:       ticketID,
		EventType:      models.TicketEventClosed,
		ActorUserID:    adminID,
		FromStatus:     statusPtr(oldStatus),
		ToStatus:       statusPtr(models.StatusClosed),
		FromEngineerID: fromEng,
		ToEngineerID:   fromEng,
		Note:           adminComment,
		CreatedAt:      now,
	})

	// Send notification for ticket closure
	if s.notificationService != nil {
		log.Printf("[ADMIN_CLOSE_TICKET] Sending closure notifications")
		go s.notificationService.NotifyTicketClosed(ticketID, adminComment)
	}

	// Resolve any pending escalations for this ticket
	if err := s.escalationRepo.ResolveByTicket(ticketID); err != nil {
		log.Printf("[ADMIN_CLOSE_TICKET_WARN] Failed to resolve escalations for ticket %s: %v", ticketID, err)
		// Don't return error, ticket is already closed
	} else {
		log.Printf("[ADMIN_CLOSE_TICKET] Escalations resolved for ticket %s", ticketID)
	}

	log.Printf("[ADMIN_CLOSE_TICKET_SUCCESS] Ticket closed successfully by admin")
	return nil
}

// =========================
// ADMIN: GET ALL TICKETS
// =========================
func (s *TicketService) GetAll() ([]models.Ticket, error) {
	return s.ticketRepo.GetAll()
}

func (s *TicketService) AdminCreateTicketAndAssign(
	customerID uuid.UUID,
	customerSolutionID uuid.UUID,
	title string,
	description string,
	engineerID uuid.UUID,
	priority models.TicketPriority,
	supportMode models.SupportMode,
	adminID uuid.UUID,
) (*models.Ticket, error) {

	var ticket *models.Ticket

	err := s.db.Transaction(func(tx *gorm.DB) error {

		cs, err := s.customerSolutionRepo.GetByID(customerSolutionID)
		if err != nil {
			return errors.New("customer solution not found")
		}

		if cs.CustomerID != customerID {
			return errors.New("solution does not belong to customer")
		}

		slaHours := 72
		if cs.ContractType == models.ContractAMC {
			slaHours = 24
		}

		targetAt := time.Now().Add(time.Duration(slaHours) * time.Hour)

		// Generate custom ticket ID
		ticketID, err := s.ticketRepo.GenerateNextTicketID()
		if err != nil {
			return errors.New("failed to generate ticket ID")
		}

		ticket = &models.Ticket{
			ID:                 ticketID,
			CustomerID:         customerID,
			CustomerSolutionID: &cs.ID,
			EngineerID:         &engineerID,
			Title:              title,
			Description:        description,
			Status:             models.StatusAssigned,
			Priority:           priority,
			SupportMode:        supportMode,
			ServiceCallType:    models.ServiceCallType(cs.ContractType),
			SLAHours:           slaHours,
			TargetAt:           &targetAt,
			CreatedBy:          adminID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if err := s.ticketRepo.CreateTx(tx, ticket); err != nil {
			return err
		}

		if err := s.ticketRepo.AssignEngineerTx(
			tx,
			ticket.ID,
			engineerID,
			adminID,
		); err != nil {
			return err
		}

		now := time.Now()
		if err := s.logTicketEventTx(tx, &models.TicketEvent{
			TicketID:    ticket.ID,
			EventType:   models.TicketEventCreated,
			ActorUserID: adminID,
			ToStatus:    statusPtr(models.StatusAssigned),
			CreatedAt:   now,
		}); err != nil {
			return err
		}

		return s.logTicketEventTx(tx, &models.TicketEvent{
			TicketID:     ticket.ID,
			EventType:    models.TicketEventAssigned,
			ActorUserID:  adminID,
			FromStatus:   statusPtr(models.StatusOpen),
			ToStatus:     statusPtr(models.StatusAssigned),
			ToEngineerID: uuidPtr(engineerID),
			CreatedAt:    now,
		})
	})

	if err != nil {
		return nil, err
	}

	return ticket, nil
}

func (s *TicketService) AdminAssignTicket(
	ticketID string,
	customerSolutionID uuid.UUID,
	engineerID uuid.UUID,
	adminID uuid.UUID,
	priority models.TicketPriority,
	supportMode models.SupportMode,
	serviceCallType models.ServiceCallType,
) error {
	log.Printf("[ADMIN_ASSIGN_TICKET] Starting - ticketID=%s engineerID=%s", ticketID, engineerID)

	err := s.db.Transaction(func(tx *gorm.DB) error {

		repo := repository.NewTicketRepository(tx)

		ticket, err := repo.GetByID(ticketID)
		if err != nil {
			log.Printf("[ADMIN_ASSIGN_TICKET_ERROR] Failed to get ticket: %v", err)
			return err
		}

		log.Printf("[ADMIN_ASSIGN_TICKET] Current status: %s", ticket.Status)

		if err := assertTicketTransition(ticket.Status, models.StatusAssigned); err != nil {
			log.Printf("[ADMIN_ASSIGN_TICKET_ERROR] %v", err)
			return err
		}

		cs, err := s.customerSolutionRepo.GetByID(customerSolutionID)
		if err != nil {
			log.Printf("[ADMIN_ASSIGN_TICKET_ERROR] Failed to get customer solution: %v", err)
			return errors.New("invalid customer solution")
		}

		// SLA logic
		slaHours := 72
		if serviceCallType == models.ServiceTypeAMC {
			slaHours = 24
		}

		targetAt := time.Now().Add(time.Duration(slaHours) * time.Hour)

		// Update ticket
		log.Printf("[ADMIN_ASSIGN_TICKET] Updating ticket status to Assigned")
		if err := repo.UpdateFields(ticketID, map[string]interface{}{
			"customer_solution_id": cs.ID,
			"engineer_id":          engineerID,
			"service_call_type":    serviceCallType,
			"priority":             priority,
			"support_mode":         supportMode,
			"sla_hours":            slaHours,
			"target_at":            targetAt,
			"status":               models.StatusAssigned,
			"updated_at":           time.Now(),
		}); err != nil {
			log.Printf("[ADMIN_ASSIGN_TICKET_ERROR] Failed to update ticket: %v", err)
			return err
		}

		log.Printf("[ADMIN_ASSIGN_TICKET] Ticket status updated successfully")

		// Assign engineer
		log.Printf("[ADMIN_ASSIGN_TICKET] Creating ticket assignment")
		if err := repo.AssignEngineer(&models.TicketAssignment{
			TicketID:           ticketID,
			CustomerSolutionID: cs.ID,
			EngineerID:         engineerID,
			AssignedBy:         adminID,
			AssignedAt:         time.Now(),
		}); err != nil {
			return err
		}

		now := time.Now()
		if err := repo.CreateStatusHistory(&models.TicketStatusHistory{
			TicketID:  ticketID,
			OldStatus: string(ticket.Status),
			NewStatus: string(models.StatusAssigned),
			ChangedBy: adminID,
			ChangedAt: now,
		}); err != nil {
			return err
		}

		return s.logTicketEventTx(tx, &models.TicketEvent{
			TicketID:     ticketID,
			EventType:    models.TicketEventAssigned,
			ActorUserID:  adminID,
			FromStatus:   statusPtr(ticket.Status),
			ToStatus:     statusPtr(models.StatusAssigned),
			ToEngineerID: uuidPtr(engineerID),
			CreatedAt:    now,
		})
	})

	if err != nil {
		log.Printf("[ADMIN_ASSIGN_TICKET_ERROR] Transaction failed: %v", err)
		return err
	}

	log.Printf("[ADMIN_ASSIGN_TICKET] Transaction committed successfully")

	// Send notification for ticket assignment
	if s.notificationService != nil {
		log.Printf("[ADMIN_ASSIGN_TICKET] Sending notifications")
		go s.notificationService.NotifyTicketAssigned(ticketID, engineerID)
	}

	log.Printf("[ADMIN_ASSIGN_TICKET_SUCCESS] Ticket assigned to engineer")
	return nil
}

/*
=========================

	GET TICKET BY ID (CUSTOMER)

=========================
*/
func (s *TicketService) GetTicketById(ticketID string, userID uuid.UUID) (*models.Ticket, error) {
	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, err
	}

	// Verify that the ticket belongs to the customer
	if ticket.CustomerID != userID {
		return nil, errors.New("ticket not found or access denied")
	}

	return ticket, nil
}

/*
=========================

	ADMIN: REASSIGN TICKET

=========================
*/
func (s *TicketService) ReassignTicket(
	ticketID string,
	newEngineerID uuid.UUID,
	adminID uuid.UUID,
) (*models.Ticket, error) {
	log.Printf("[REASSIGN_TICKET] Starting - ticketID=%s newEngineerID=%s adminID=%s", ticketID, newEngineerID, adminID)

	var updatedTicket *models.Ticket

	err := s.db.Transaction(func(tx *gorm.DB) error {
		repo := repository.NewTicketRepository(tx)

		// Get the ticket
		ticket, err := repo.GetByID(ticketID)
		if err != nil {
			log.Printf("[REASSIGN_TICKET_ERROR] Failed to get ticket: %v", err)
			return err
		}

		// Verify ticket is assigned and status allows reassignment
		if ticket.Status == models.StatusClosed {
			log.Printf("[REASSIGN_TICKET_ERROR] Cannot reassign closed ticket")
			return errors.New("cannot reassign a closed ticket")
		}

		if ticket.EngineerID == nil {
			log.Printf("[REASSIGN_TICKET_ERROR] Ticket is not assigned")
			return errors.New("ticket is not assigned to any engineer")
		}

		// Prevent reassigning to the same engineer
		if *ticket.EngineerID == newEngineerID {
			log.Printf("[REASSIGN_TICKET_ERROR] Cannot reassign to same engineer")
			return errors.New("please select a different engineer")
		}

		// Verify new engineer exists
		seRepo := repository.NewSupportEngineerRepository(tx)
		_, err = seRepo.GetByID(newEngineerID)
		if err != nil {
			log.Printf("[REASSIGN_TICKET_ERROR] Engineer not found: %v", err)
			return errors.New("selected engineer not found")
		}

		// Update the ticket with new engineer
		log.Printf("[REASSIGN_TICKET] Updating ticket assignment from %s to %s", ticket.EngineerID, newEngineerID)
		if err := repo.UpdateFields(ticketID, map[string]interface{}{
			"engineer_id": newEngineerID,
			"updated_at":  time.Now(),
		}); err != nil {
			log.Printf("[REASSIGN_TICKET_ERROR] Failed to update ticket: %v", err)
			return err
		}

		// Create new assignment record
		log.Printf("[REASSIGN_TICKET] Creating new assignment record")
		oldEngineerID := *ticket.EngineerID
		csID := uuid.Nil
		if ticket.CustomerSolutionID != nil {
			csID = *ticket.CustomerSolutionID
		}
		if err := repo.AssignEngineer(&models.TicketAssignment{
			TicketID:           ticketID,
			CustomerSolutionID: csID,
			EngineerID:         newEngineerID,
			AssignedBy:         adminID,
			AssignedAt:         time.Now(),
		}); err != nil {
			log.Printf("[REASSIGN_TICKET_ERROR] Failed to create assignment: %v", err)
			return err
		}

		if err := s.logTicketEventTx(tx, &models.TicketEvent{
			TicketID:       ticketID,
			EventType:      models.TicketEventReassigned,
			ActorUserID:    adminID,
			FromStatus:     statusPtr(ticket.Status),
			ToStatus:       statusPtr(ticket.Status),
			FromEngineerID: uuidPtr(oldEngineerID),
			ToEngineerID:   uuidPtr(newEngineerID),
			CreatedAt:      time.Now(),
		}); err != nil {
			return err
		}

		// Fetch updated ticket
		updatedTicket, err = repo.GetByID(ticketID)
		if err != nil {
			log.Printf("[REASSIGN_TICKET_ERROR] Failed to fetch updated ticket: %v", err)
			return err
		}

		log.Printf("[REASSIGN_TICKET] Assignment record created successfully")
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Send notifications to old and new engineer
	if s.notificationService != nil {
		log.Printf("[REASSIGN_TICKET] Sending notifications")
		go s.notificationService.NotifyTicketAssigned(ticketID, newEngineerID)
	}

	log.Printf("[REASSIGN_TICKET_SUCCESS] Ticket reassigned successfully")
	return updatedTicket, nil
}

/* =========================
   FIELD VISITS
========================= */

type CreateFieldVisitInput struct {
	TicketID       string
	VisitDate      time.Time
	Notes          string
	CoEngineerIDs  []uuid.UUID
	ProofURLs      []string
}

func (s *TicketService) AttachVisitCounts(tickets []models.Ticket) error {
	if len(tickets) == 0 {
		return nil
	}
	ids := make([]string, 0, len(tickets))
	for _, t := range tickets {
		ids = append(ids, t.ID)
	}
	counts, err := s.visitRepo.CountByTicketIDs(ids)
	if err != nil {
		return err
	}
	for i := range tickets {
		tickets[i].VisitCount = counts[tickets[i].ID]
	}
	return nil
}

func (s *TicketService) GetAllWithVisitCounts() ([]models.Ticket, error) {
	tickets, err := s.ticketRepo.GetAll()
	if err != nil {
		return nil, err
	}
	if err := s.AttachVisitCounts(tickets); err != nil {
		log.Printf("[GET_ADMIN_TICKETS] visit count attach failed: %v", err)
	}
	return tickets, nil
}

func (s *TicketService) ListFieldVisits(ticketID string) ([]models.ServiceVisit, error) {
	if ticketID == "" {
		return nil, errors.New("ticket_id is required")
	}
	if _, err := s.ticketRepo.GetByID(ticketID); err != nil {
		return nil, errors.New("ticket not found")
	}
	return s.visitRepo.ListByTicketID(ticketID)
}

func (s *TicketService) ListAllFieldVisits(
	engineerID *uuid.UUID,
	companyID *uuid.UUID,
	startDate *time.Time,
	endDate *time.Time,
) ([]models.ServiceVisit, error) {
	return s.visitRepo.ListAll(repository.ListAllVisitsFilter{
		EngineerID: engineerID,
		CompanyID:  companyID,
		StartDate:  startDate,
		EndDate:    endDate,
	})
}

func (s *TicketService) ListFieldVisitsForAssignedEngineer(
	ticketID string,
	userID uuid.UUID,
) ([]models.ServiceVisit, error) {
	engineer, err := s.resolveSupportEngineer(userID)
	if err != nil {
		return nil, err
	}

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}
	if ticket.EngineerID == nil || *ticket.EngineerID != engineer.ID {
		return nil, errors.New("you are not assigned to this ticket")
	}

	return s.visitRepo.ListByTicketID(ticketID)
}

func (s *TicketService) CreateFieldVisit(
	userID uuid.UUID,
	input CreateFieldVisitInput,
) (*models.ServiceVisit, error) {
	if input.TicketID == "" {
		return nil, errors.New("ticket_id is required")
	}
	if input.Notes == "" {
		return nil, errors.New("notes are required")
	}
	if input.VisitDate.IsZero() {
		return nil, errors.New("visit_date is required")
	}

	engineer, err := s.resolveSupportEngineer(userID)
	if err != nil {
		return nil, err
	}

	ticket, err := s.ticketRepo.GetByID(input.TicketID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}
	if ticket.EngineerID == nil || *ticket.EngineerID != engineer.ID {
		return nil, errors.New("you are not assigned to this ticket")
	}
	if ticket.Status != models.StatusInProgress {
		return nil, errors.New("field visits can only be logged while ticket is In Progress")
	}

	// Validate co-engineers exist and exclude self
	coIDs := make([]uuid.UUID, 0, len(input.CoEngineerIDs))
	seen := map[uuid.UUID]bool{}
	for _, id := range input.CoEngineerIDs {
		if id == uuid.Nil || id == engineer.ID || seen[id] {
			continue
		}
		var eng models.SupportEngineer
		if err := s.db.Where("id = ? AND is_active = true", id).First(&eng).Error; err != nil {
			return nil, fmt.Errorf("invalid co-engineer id: %s", id)
		}
		seen[id] = true
		coIDs = append(coIDs, id)
	}

	visitDate := time.Date(
		input.VisitDate.Year(),
		input.VisitDate.Month(),
		input.VisitDate.Day(),
		0, 0, 0, 0,
		time.UTC,
	)

	visit := &models.ServiceVisit{
		TicketID:   input.TicketID,
		EngineerID: engineer.ID,
		VisitDate:  visitDate,
		Notes:      input.Notes,
		StartTime:  &visitDate, // satisfy legacy NOT NULL column if present
	}

	if err := s.visitRepo.Create(visit); err != nil {
		return nil, err
	}

	if err := s.visitRepo.ReplaceCoEngineers(visit.ID, coIDs); err != nil {
		return nil, err
	}

	proofs := make([]models.ServiceVisitProof, 0, len(input.ProofURLs))
	for _, url := range input.ProofURLs {
		if url == "" {
			continue
		}
		proofs = append(proofs, models.ServiceVisitProof{
			ServiceVisitID: visit.ID,
			URL:            url,
		})
	}
	if err := s.visitRepo.CreateProofs(proofs); err != nil {
		return nil, err
	}

	visits, err := s.visitRepo.ListByTicketID(input.TicketID)
	if err != nil {
		return visit, nil
	}
	for i := range visits {
		if visits[i].ID == visit.ID {
			return &visits[i], nil
		}
	}
	return visit, nil
}

func (s *TicketService) resolveSupportEngineer(userID uuid.UUID) (*models.SupportEngineer, error) {
	var engineer models.SupportEngineer
	if err := s.db.Where("user_id = ?", userID).First(&engineer).Error; err != nil {
		return nil, errors.New("support engineer profile not found")
	}
	return &engineer, nil
}

/* =========================
   ADMIN: REOPEN TICKET
========================= */

func (s *TicketService) ReopenTicket(
	ticketID string,
	adminID uuid.UUID,
	note string,
) (*models.Ticket, error) {
	log.Printf("[REOPEN_TICKET] ticketID=%s adminID=%s", ticketID, adminID)

	ticket, err := s.ticketRepo.GetByID(ticketID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if err := assertTicketTransition(ticket.Status, models.StatusOpen); err != nil {
		return nil, err
	}

	now := time.Now()
	var fromEng *uuid.UUID
	if ticket.EngineerID != nil {
		fromEng = ticket.EngineerID
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		repo := repository.NewTicketRepository(tx)

		if err := repo.UpdateFields(ticketID, map[string]interface{}{
			"status":               models.StatusOpen,
			"engineer_id":          nil,
			"customer_solution_id": nil,
			"closed_at":            nil,
			"support_comment":      nil,
			"closure_proof_image":  nil,
			"updated_at":           now,
		}); err != nil {
			return err
		}

		if err := repo.CreateStatusHistory(&models.TicketStatusHistory{
			TicketID:  ticketID,
			OldStatus: string(models.StatusClosed),
			NewStatus: string(models.StatusOpen),
			ChangedBy: adminID,
			ChangedAt: now,
		}); err != nil {
			return err
		}

		return s.logTicketEventTx(tx, &models.TicketEvent{
			TicketID:       ticketID,
			EventType:      models.TicketEventReopened,
			ActorUserID:    adminID,
			FromStatus:     statusPtr(models.StatusClosed),
			ToStatus:       statusPtr(models.StatusOpen),
			FromEngineerID: fromEng,
			Note:           note,
			CreatedAt:      now,
		})
	})
	if err != nil {
		return nil, err
	}

	return s.ticketRepo.GetByID(ticketID)
}

/* =========================
   ADMIN: TICKET STATUS PAGE
========================= */

type TicketStatusRow struct {
	Ticket           models.Ticket       `json:"ticket"`
	LastEvent        *models.TicketEvent `json:"last_event,omitempty"`
	ReopenCount      int64               `json:"reopen_count"`
	CompanyName      string              `json:"company_name"`
	CustomerName     string              `json:"customer_name"`
	EngineerName     string              `json:"engineer_name"`
	LastEventSummary string              `json:"last_event_summary"`
}

func engineerDisplayName(eng *models.SupportEngineer) string {
	if eng == nil {
		return ""
	}
	if eng.User.Name != "" {
		return eng.User.Name
	}
	if eng.User.Email != "" {
		return eng.User.Email
	}
	return ""
}

func eventSummary(ev *models.TicketEvent) string {
	if ev == nil {
		return ""
	}
	switch ev.EventType {
	case models.TicketEventCreated:
		return "Created"
	case models.TicketEventAssigned:
		name := engineerDisplayName(ev.ToEngineer)
		if name != "" {
			return "Assigned to " + name
		}
		return "Assigned"
	case models.TicketEventReassigned:
		from := engineerDisplayName(ev.FromEngineer)
		to := engineerDisplayName(ev.ToEngineer)
		if from != "" && to != "" {
			return "Reassigned from " + from + " to " + to
		}
		if to != "" {
			return "Reassigned to " + to
		}
		return "Reassigned"
	case models.TicketEventStarted:
		return "Started"
	case models.TicketEventClosed:
		return "Closed"
	case models.TicketEventReopened:
		return "Reopened"
	default:
		return string(ev.EventType)
	}
}

func (s *TicketService) ListTicketStatus(filter repository.TicketStatusListFilter) ([]TicketStatusRow, error) {
	tickets, err := s.ticketRepo.ListForStatusPage(filter)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(tickets))
	for _, t := range tickets {
		ids = append(ids, t.ID)
	}

	latest, err := s.ticketRepo.GetLatestEventsByTicketIDs(ids)
	if err != nil {
		log.Printf("[TICKET_STATUS] latest events failed: %v", err)
		latest = map[string]models.TicketEvent{}
	}
	reopens, err := s.ticketRepo.CountReopensByTicketIDs(ids)
	if err != nil {
		log.Printf("[TICKET_STATUS] reopen counts failed: %v", err)
		reopens = map[string]int64{}
	}

	rows := make([]TicketStatusRow, 0, len(tickets))
	for _, t := range tickets {
		row := TicketStatusRow{Ticket: t, ReopenCount: reopens[t.ID]}
		row.CustomerName = t.Customer.Name
		if t.Customer.Company.Name != "" {
			row.CompanyName = t.Customer.Company.Name
		}
		row.EngineerName = engineerDisplayName(t.SupportEngineer)
		if ev, ok := latest[t.ID]; ok {
			copyEv := ev
			row.LastEvent = &copyEv
			row.LastEventSummary = eventSummary(&copyEv)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *TicketService) ListTicketEvents(ticketID string) ([]models.TicketEvent, error) {
	if ticketID == "" {
		return nil, errors.New("ticket_id is required")
	}
	if _, err := s.ticketRepo.GetByID(ticketID); err != nil {
		return nil, errors.New("ticket not found")
	}
	return s.ticketRepo.ListEventsByTicketID(ticketID)
}
