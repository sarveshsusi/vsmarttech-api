package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/domain"
	"rbac/models"
	"rbac/repository"
)

type AssetDropService struct {
	db                  *gorm.DB
	dropRepo            *repository.AssetDropRepository
	ticketRepo          *repository.TicketRepository
	assetRepo           *repository.AssetRepository
	supportEngineerRepo *repository.SupportEngineerRepository
	notificationService *NotificationService
}

func NewAssetDropService(
	db *gorm.DB,
	dropRepo *repository.AssetDropRepository,
	ticketRepo *repository.TicketRepository,
	assetRepo *repository.AssetRepository,
	supportEngineerRepo *repository.SupportEngineerRepository,
	notificationService *NotificationService,
) *AssetDropService {
	return &AssetDropService{
		db:                  db,
		dropRepo:            dropRepo,
		ticketRepo:          ticketRepo,
		assetRepo:           assetRepo,
		supportEngineerRepo: supportEngineerRepo,
		notificationService: notificationService,
	}
}

type CreateAssetDropInput struct {
	TicketID      string
	SerialNumber  string
	Name          string
	Model         string
	Category      string
	Site          string
	Location      string
	IsReplacement bool
}

func joinSiteLocation(site, location string) string {
	site = strings.TrimSpace(site)
	location = strings.TrimSpace(location)
	switch {
	case site != "" && location != "":
		return site + " / " + location
	case site != "":
		return site
	default:
		return location
	}
}

func (s *AssetDropService) AttachLatestToTickets(tickets []models.Ticket) {
	if len(tickets) == 0 {
		return
	}
	ids := make([]string, 0, len(tickets))
	for i := range tickets {
		ids = append(ids, tickets[i].ID)
	}
	byTicket, err := s.dropRepo.ListLatestByTicketIDs(ids)
	if err != nil {
		return
	}
	for i := range tickets {
		if d, ok := byTicket[tickets[i].ID]; ok {
			cp := d
			tickets[i].AssetDrop = &cp
		}
	}
}

func (s *AssetDropService) AttachLatestToTicket(ticket *models.Ticket) {
	if ticket == nil {
		return
	}
	drop, err := s.dropRepo.GetLatestByTicketID(ticket.ID)
	if err != nil {
		return
	}
	ticket.AssetDrop = drop
}

func (s *AssetDropService) ListAdmin(statuses []models.AssetDropStatus) ([]models.TicketAssetDrop, error) {
	return s.dropRepo.ListAdmin(statuses)
}

func (s *AssetDropService) ListForSupportUser(userID uuid.UUID) ([]models.TicketAssetDrop, error) {
	engineer, err := s.supportEngineerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("support engineer profile not found")
	}
	return s.dropRepo.ListForSupportEngineer(engineer.ID)
}

func (s *AssetDropService) ListReturnAssignments(userID uuid.UUID) ([]models.TicketAssetDrop, error) {
	engineer, err := s.supportEngineerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("support engineer profile not found")
	}
	return s.dropRepo.ListReturnAssignments(engineer.ID)
}

func (s *AssetDropService) pauseSLA(ticket *models.Ticket, now time.Time) map[string]interface{} {
	fields := map[string]interface{}{
		"updated_at": now,
	}
	if ticket.SLAPausedAt == nil {
		fields["sla_paused_at"] = now
	}
	return fields
}

// resumeSLAFields clears the pause and extends target_at so remaining SLA is unchanged.
// GORM Updates skips bare nil map values — use Expr("NULL") so sla_paused_at is cleared.
func (s *AssetDropService) resumeSLAFields(ticket *models.Ticket, now time.Time) map[string]interface{} {
	fields := map[string]interface{}{
		"sla_paused_at": gorm.Expr("NULL"),
		"updated_at":    now,
	}
	if ticket.SLAPausedAt == nil {
		return fields
	}
	elapsed := int(now.Sub(*ticket.SLAPausedAt).Seconds())
	if elapsed < 0 {
		elapsed = 0
	}
	fields["sla_paused_total_seconds"] = ticket.SLAPausedTotalSeconds + elapsed
	if ticket.TargetAt != nil {
		extended := ticket.TargetAt.Add(time.Duration(elapsed) * time.Second)
		fields["target_at"] = extended
	}
	return fields
}

// resumeHaltedTicket moves Halted → In Progress and unfreezes SLA inside an existing tx.
func (s *AssetDropService) resumeHaltedTicket(
	tx *gorm.DB,
	ticket *models.Ticket,
	adminUserID uuid.UUID,
	note string,
	now time.Time,
) error {
	if ticket.Status != models.StatusHalted {
		return nil
	}
	if !domain.CanTransition(ticket.Status, models.StatusInProgress) {
		return fmt.Errorf("invalid status transition from %s to In Progress", ticket.Status)
	}
	ticketRepo := repository.NewTicketRepository(tx)
	fields := s.resumeSLAFields(ticket, now)
	fields["status"] = models.StatusInProgress
	if err := ticketRepo.UpdateFields(ticket.ID, fields); err != nil {
		return err
	}
	if err := ticketRepo.CreateStatusHistory(&models.TicketStatusHistory{
		TicketID:  ticket.ID,
		OldStatus: string(models.StatusHalted),
		NewStatus: string(models.StatusInProgress),
		ChangedBy: adminUserID,
		ChangedAt: now,
	}); err != nil {
		return err
	}
	return ticketRepo.CreateEventTx(tx, &models.TicketEvent{
		TicketID:    ticket.ID,
		EventType:   models.TicketEventResumed,
		ActorUserID: adminUserID,
		FromStatus:  statusPtr(models.StatusHalted),
		ToStatus:    statusPtr(models.StatusInProgress),
		Note:        note,
		CreatedAt:   now,
	})
}

// RequestAssetDrop is called by the assigned support engineer.
func (s *AssetDropService) RequestAssetDrop(userID uuid.UUID, in CreateAssetDropInput) (*models.TicketAssetDrop, error) {
	serial := strings.TrimSpace(in.SerialNumber)
	name := strings.TrimSpace(in.Name)
	if in.TicketID == "" || serial == "" || name == "" {
		return nil, errors.New("ticket_id, serial_number, and name are required")
	}

	engineer, err := s.supportEngineerRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("support engineer profile not found")
	}

	ticket, err := s.ticketRepo.GetByID(in.TicketID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}
	if ticket.EngineerID == nil || *ticket.EngineerID != engineer.ID {
		return nil, errors.New("you are not assigned to this ticket")
	}
	if ticket.Status != models.StatusInProgress {
		return nil, fmt.Errorf("asset drop can only be requested after starting the ticket (current: %s)", ticket.Status)
	}
	if err := assertTicketTransition(ticket.Status, models.StatusHalted); err != nil {
		return nil, err
	}

	if _, err := s.dropRepo.GetActiveByTicketID(ticket.ID); err == nil {
		return nil, errors.New("an active asset drop already exists for this ticket")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if existing, err := s.assetRepo.FindBySerialNumber(serial); err == nil && existing != nil {
		return nil, errors.New("an asset with this serial number already exists")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	fromStatus := ticket.Status
	drop := &models.TicketAssetDrop{
		TicketID:      ticket.ID,
		SerialNumber:  serial,
		Name:          name,
		Model:         strings.TrimSpace(in.Model),
		Category:      strings.TrimSpace(in.Category),
		SiteLocation:  joinSiteLocation(in.Site, in.Location),
		IsReplacement: in.IsReplacement,
		Status:        models.AssetDropStatusRequested,
		RequestedBy:   userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		dropRepo := repository.NewAssetDropRepository(tx)
		ticketRepo := repository.NewTicketRepository(tx)

		if err := dropRepo.Create(drop); err != nil {
			return err
		}

		fields := s.pauseSLA(ticket, now)
		fields["status"] = models.StatusHalted
		if err := ticketRepo.UpdateFields(ticket.ID, fields); err != nil {
			return err
		}

		if err := ticketRepo.CreateStatusHistory(&models.TicketStatusHistory{
			TicketID:  ticket.ID,
			OldStatus: string(fromStatus),
			NewStatus: string(models.StatusHalted),
			ChangedBy: engineer.ID,
			ChangedAt: now,
		}); err != nil {
			return err
		}

		return repository.NewTicketRepository(tx).CreateEventTx(tx, &models.TicketEvent{
			TicketID:       ticket.ID,
			EventType:      models.TicketEventHalted,
			ActorUserID:    userID,
			FromStatus:     statusPtr(fromStatus),
			ToStatus:       statusPtr(models.StatusHalted),
			FromEngineerID: uuidPtr(engineer.ID),
			ToEngineerID:   uuidPtr(engineer.ID),
			Note:           "Asset drop requested: " + serial,
			CreatedAt:      now,
		})
	})
	if err != nil {
		return nil, err
	}

	if s.notificationService != nil {
		go s.notificationService.NotifyAssetDropRequested(ticket.ID, drop.ID, serial, name)
	}

	created, err := s.dropRepo.GetByID(drop.ID)
	if err != nil {
		return drop, nil
	}
	return created, nil
}

func (s *AssetDropService) Acknowledge(adminUserID uuid.UUID, dropID uuid.UUID) (*models.TicketAssetDrop, error) {
	drop, err := s.dropRepo.GetByID(dropID)
	if err != nil {
		return nil, errors.New("asset drop not found")
	}
	if drop.Status != models.AssetDropStatusRequested {
		return nil, fmt.Errorf("drop must be in requested status (current: %s)", drop.Status)
	}

	ticket, err := s.ticketRepo.GetByID(drop.TicketID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}
	if ticket.Customer.CompanyID == uuid.Nil {
		// Ensure company is loaded
		ticket, err = s.ticketRepo.GetByID(drop.TicketID)
		if err != nil {
			return nil, errors.New("ticket not found")
		}
	}
	companyID := ticket.Customer.CompanyID
	if companyID == uuid.Nil {
		return nil, errors.New("ticket customer has no company")
	}

	if existing, err := s.assetRepo.FindBySerialNumber(drop.SerialNumber); err == nil && existing != nil {
		return nil, errors.New("an asset with this serial number already exists")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	var assetID uuid.UUID

	err = s.db.Transaction(func(tx *gorm.DB) error {
		assetRepo := repository.NewAssetRepository(tx)
		dropRepo := repository.NewAssetDropRepository(tx)
		ticketRepo := repository.NewTicketRepository(tx)

		customerID := ticket.CustomerID
		asset := &models.Asset{
			CompanyID:          companyID,
			CustomerID:         &customerID,
			CustomerSolutionID: ticket.CustomerSolutionID,
			SerialNumber:       drop.SerialNumber,
			Name:               drop.Name,
			Model:              drop.Model,
			Category:           drop.Category,
			SiteLocation:       drop.SiteLocation,
			Status:             models.AssetStatusCollected,
			IsReplacement:      drop.IsReplacement,
			CreatedBy:          adminUserID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := assetRepo.Create(asset); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
				strings.Contains(strings.ToLower(err.Error()), "unique") {
				return errors.New("an asset with this serial number already exists")
			}
			return err
		}
		assetID = asset.ID

		ticketID := ticket.ID
		_ = assetRepo.CreateStatusHistory(&models.AssetStatusHistory{
			AssetID:   asset.ID,
			OldStatus: "",
			NewStatus: models.AssetStatusCollected,
			TicketID:  &ticketID,
			ChangedBy: adminUserID,
			ChangedAt: now,
		})

		if err := ticketRepo.UpdateFields(ticket.ID, map[string]interface{}{
			"asset_id":   asset.ID,
			"updated_at": now,
		}); err != nil {
			return err
		}

		drop.Status = models.AssetDropStatusAcknowledged
		drop.AssetID = &asset.ID
		drop.AcknowledgedBy = &adminUserID
		drop.AcknowledgedAt = &now
		drop.UpdatedAt = now
		return dropRepo.Update(drop)
	})
	if err != nil {
		return nil, err
	}

	_ = assetID
	if s.notificationService != nil {
		go s.notificationService.NotifyAssetDropAcknowledged(ticket.ID, drop.ID, drop.RequestedBy, drop.SerialNumber)
	}

	return s.dropRepo.GetByID(drop.ID)
}

func (s *AssetDropService) AssignReturnEngineer(adminUserID uuid.UUID, dropID, engineerID uuid.UUID) (*models.TicketAssetDrop, error) {
	drop, err := s.dropRepo.GetByID(dropID)
	if err != nil {
		return nil, errors.New("asset drop not found")
	}
	if drop.Status != models.AssetDropStatusAcknowledged && drop.Status != models.AssetDropStatusReturnAssigned {
		return nil, fmt.Errorf("return engineer can only be assigned after acknowledgement (current: %s)", drop.Status)
	}

	exists, err := s.ticketRepo.SupportEngineerExists(engineerID)
	if err != nil || !exists {
		return nil, errors.New("support engineer not found")
	}

	now := time.Now()
	drop.ReturnEngineerID = &engineerID
	drop.Status = models.AssetDropStatusReturnAssigned
	drop.ReturnAssignedAt = &now
	drop.UpdatedAt = now
	if err := s.dropRepo.Update(drop); err != nil {
		return nil, err
	}

	if s.notificationService != nil {
		go s.notificationService.NotifyAssetDropReturnAssigned(drop.TicketID, drop.ID, engineerID, drop.SerialNumber)
	}

	_ = adminUserID
	return s.dropRepo.GetByID(drop.ID)
}

func (s *AssetDropService) SendToSite(adminUserID uuid.UUID, dropID uuid.UUID) (*models.TicketAssetDrop, error) {
	drop, err := s.dropRepo.GetByID(dropID)
	if err != nil {
		return nil, errors.New("asset drop not found")
	}
	// Allow resume if already returned but ticket still Halted (stuck state).
	if drop.Status != models.AssetDropStatusReturnAssigned &&
		drop.Status != models.AssetDropStatusAcknowledged &&
		drop.Status != models.AssetDropStatusReturned {
		return nil, errors.New("assign a return engineer before sending to site")
	}
	if drop.Status == models.AssetDropStatusAcknowledged {
		return nil, errors.New("assign a return engineer before sending to site")
	}
	if drop.AssetID == nil {
		return nil, errors.New("asset drop has no linked asset")
	}

	ticket, err := s.ticketRepo.GetByID(drop.TicketID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	asset, err := s.assetRepo.GetByID(*drop.AssetID)
	if err != nil {
		return nil, errors.New("linked asset not found")
	}

	now := time.Now()
	oldAssetStatus := asset.Status
	alreadyReturned := models.NormalizeAssetStatus(asset.Status) == models.AssetStatusReturnedToSite

	err = s.db.Transaction(func(tx *gorm.DB) error {
		assetRepo := repository.NewAssetRepository(tx)
		dropRepo := repository.NewAssetDropRepository(tx)

		if !alreadyReturned {
			asset.Status = models.AssetStatusReturnedToSite
			asset.UpdatedAt = now
			if err := assetRepo.Update(asset); err != nil {
				return err
			}
			ticketID := ticket.ID
			_ = assetRepo.CreateStatusHistory(&models.AssetStatusHistory{
				AssetID:   asset.ID,
				OldStatus: oldAssetStatus,
				NewStatus: models.AssetStatusReturnedToSite,
				TicketID:  &ticketID,
				ChangedBy: adminUserID,
				ChangedAt: now,
			})
		}

		if err := s.resumeHaltedTicket(
			tx,
			ticket,
			adminUserID,
			"Asset sent to site: "+drop.SerialNumber,
			now,
		); err != nil {
			return err
		}

		if drop.Status != models.AssetDropStatusReturned {
			drop.Status = models.AssetDropStatusReturned
			drop.ReturnedAt = &now
			drop.UpdatedAt = now
			return dropRepo.Update(drop)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.notificationService != nil {
		go s.notificationService.NotifyAssetDropReturned(ticket.ID, drop.ID, drop.SerialNumber)
	}

	return s.dropRepo.GetByID(drop.ID)
}

// CompleteReturnForAsset resumes a halted ticket when workshop marks the device
// Returned to Site (same outcome as Send to site).
func (s *AssetDropService) CompleteReturnForAsset(adminUserID, assetID uuid.UUID) (*models.TicketAssetDrop, error) {
	drop, err := s.dropRepo.GetActiveByAssetID(assetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Maybe drop already marked returned but ticket still Halted.
			latest, lerr := s.dropRepo.GetLatestByAssetID(assetID)
			if lerr != nil {
				return nil, nil
			}
			ticket, terr := s.ticketRepo.GetByID(latest.TicketID)
			if terr != nil || ticket.Status != models.StatusHalted {
				return nil, nil
			}
			return s.SendToSite(adminUserID, latest.ID)
		}
		return nil, err
	}

	// Auto-assign return engineer gate: if still only acknowledged, require assign first
	// unless we allow completing from workshop. User flow: workshop Returned to Site should
	// un-halt. Promote acknowledged → return_assigned with ticket engineer if needed.
	if drop.Status == models.AssetDropStatusRequested {
		return nil, nil // not yet in workshop
	}

	if drop.Status == models.AssetDropStatusAcknowledged {
		ticket, terr := s.ticketRepo.GetByID(drop.TicketID)
		if terr != nil {
			return nil, terr
		}
		if ticket.EngineerID != nil {
			now := time.Now()
			drop.ReturnEngineerID = ticket.EngineerID
			drop.Status = models.AssetDropStatusReturnAssigned
			drop.ReturnAssignedAt = &now
			drop.UpdatedAt = now
			if err := s.dropRepo.Update(drop); err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("assign a return engineer before returning to site")
		}
	}

	return s.SendToSite(adminUserID, drop.ID)
}
