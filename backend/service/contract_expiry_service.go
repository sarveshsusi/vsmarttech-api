package service

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"rbac/models"
	"rbac/repository"
	"rbac/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContractExpiryService struct {
	db               *gorm.DB
	customerSolRepo  *repository.CustomerSolutionRepository
	customerRepo     *repository.CustomerRepository
	authRepo         *repository.AuthRepository
	notifRepo        *repository.NotificationRepository
	mailer           *utils.Mailer
	dashboardBaseURL string
}

func NewContractExpiryService(
	db *gorm.DB,
	customerSolRepo *repository.CustomerSolutionRepository,
	customerRepo *repository.CustomerRepository,
	authRepo *repository.AuthRepository,
	notifRepo *repository.NotificationRepository,
	mailer *utils.Mailer,
	dashboardBaseURL string,
) *ContractExpiryService {
	return &ContractExpiryService{
		db:               db,
		customerSolRepo:  customerSolRepo,
		customerRepo:     customerRepo,
		authRepo:         authRepo,
		notifRepo:        notifRepo,
		mailer:           mailer,
		dashboardBaseURL: dashboardBaseURL,
	}
}

/* =========================
   CHECK AND NOTIFY EXPIRING CONTRACTS
========================= */

// CheckAndNotifyExpiringContracts checks for contracts expiring at specific intervals
// Sends notifications ONLY at 3 months, 1 month, and 7 days before expiry
// NO daily notifications sent to customers or admins
func (s *ContractExpiryService) CheckAndNotifyExpiringContracts() {
	log.Println("[CONTRACT_EXPIRY] Starting contract expiry check...")

	// Process AMC contracts
	s.processAMCNotifications()

	// Process Warranty contracts
	s.processWarrantyNotifications()

	// Note: Admin daily summary notification has been removed
	// Only customers receive 3 specific milestone notifications: 3 months, 1 month, 7 days before expiry

	log.Println("[CONTRACT_EXPIRY] Contract expiry check completed")
}

func (s *ContractExpiryService) processAMCNotifications() {
	// Get all AMC contracts expiring within 90 days
	contracts, err := s.customerSolRepo.GetExpiringAMCs(90)
	if err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to get expiring AMCs: %v", err)
		return
	}

	log.Printf("[CONTRACT_EXPIRY] Found %d AMC contracts expiring within 90 days", len(contracts))

	for _, contract := range contracts {
		s.processAMCContract(contract)
	}
}

func (s *ContractExpiryService) processAMCContract(contract models.CustomerSolution) {
	if contract.AMCEndDate == nil {
		return
	}

	// Calculate actual days remaining
	daysRemaining := int(time.Until(*contract.AMCEndDate).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	// Only send notifications at exactly 3 specific milestones
	// Check daysRemaining: if it's within the milestone window, send the notification

	// 7 days milestone (including the day of expiry - 0 days)
	if daysRemaining <= 7 && daysRemaining >= 0 {
		notifType := models.NotificationTypeAMCExpiry7Days
		if !s.hasNotificationBeenSent(contract.ID, notifType) {
			s.sendAMCExpiryNotification(contract, 7)
		}
		return
	}

	// 30 days milestone
	if daysRemaining <= 30 && daysRemaining > 7 {
		notifType := models.NotificationTypeAMCExpiry1Month
		if !s.hasNotificationBeenSent(contract.ID, notifType) {
			s.sendAMCExpiryNotification(contract, 30)
		}
		return
	}

	// 90 days milestone
	if daysRemaining <= 90 && daysRemaining > 30 {
		notifType := models.NotificationTypeAMCExpiry3Months
		if !s.hasNotificationBeenSent(contract.ID, notifType) {
			s.sendAMCExpiryNotification(contract, 90)
		}
		return
	}
}

func (s *ContractExpiryService) processWarrantyNotifications() {
	// Get all Warranty contracts expiring within 90 days
	contracts, err := s.customerSolRepo.GetExpiringWarranties(90)
	if err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to get expiring warranties: %v", err)
		return
	}

	log.Printf("[CONTRACT_EXPIRY] Found %d Warranty contracts expiring within 90 days", len(contracts))

	for _, contract := range contracts {
		s.processWarrantyContract(contract)
	}
}

func (s *ContractExpiryService) processWarrantyContract(contract models.CustomerSolution) {
	if contract.WarrantyEndDate == nil {
		return
	}

	// Calculate actual days remaining
	daysRemaining := int(time.Until(*contract.WarrantyEndDate).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	// Only send notifications at exactly 3 specific milestones

	// 7 days milestone (including the day of expiry - 0 days)
	if daysRemaining <= 7 && daysRemaining >= 0 {
		notifType := models.NotificationTypeWarrantyExpiry7Days
		if !s.hasNotificationBeenSent(contract.ID, notifType) {
			s.sendWarrantyExpiryNotification(contract, 7)
		}
		return
	}

	// 30 days milestone
	if daysRemaining <= 30 && daysRemaining > 7 {
		notifType := models.NotificationTypeWarrantyExpiry1Month
		if !s.hasNotificationBeenSent(contract.ID, notifType) {
			s.sendWarrantyExpiryNotification(contract, 30)
		}
		return
	}

	// 90 days milestone
	if daysRemaining <= 90 && daysRemaining > 30 {
		notifType := models.NotificationTypeWarrantyExpiry3Months
		if !s.hasNotificationBeenSent(contract.ID, notifType) {
			s.sendWarrantyExpiryNotification(contract, 90)
		}
		return
	}
}

func (s *ContractExpiryService) sendAMCExpiryNotification(contract models.CustomerSolution, daysUntilExpiry int) {
	// Get customer details
	customer, err := s.customerRepo.GetByID(contract.CustomerID)
	if err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to get customer %s: %v", contract.CustomerID, err)
		return
	}

	// Determine notification type
	var notifType models.NotificationType
	switch daysUntilExpiry {
	case 90:
		notifType = models.NotificationTypeAMCExpiry3Months
	case 30:
		notifType = models.NotificationTypeAMCExpiry1Month
	case 7:
		notifType = models.NotificationTypeAMCExpiry7Days
	}

	solutionName := "Unknown Solution"
	if contract.Solution.Title != "" {
		solutionName = contract.Solution.Title
	}

	expiryDate := ""
	if contract.AMCEndDate != nil {
		expiryDate = contract.AMCEndDate.Format("02 Jan 2006")
	}

	// Calculate actual days remaining for display
	actualDaysRemaining := 0
	if contract.AMCEndDate != nil {
		actualDaysRemaining = int(time.Until(*contract.AMCEndDate).Hours() / 24)
		if actualDaysRemaining < 0 {
			actualDaysRemaining = 0
		}
	}

	title := fmt.Sprintf("AMC Expiring Soon - %d Days Remaining", actualDaysRemaining)
	message := fmt.Sprintf("Your AMC for %s (PO: %s) expires on %s. Please renew to continue support services.",
		solutionName, contract.PONumber, expiryDate)

	// Create in-app notification for the customer user
	notification := &models.Notification{
		ID:                 uuid.New(),
		UserID:             customer.UserID,
		CustomerSolutionID: &contract.ID,
		Type:               notifType,
		Title:              title,
		Message:            message,
		IsRead:             false,
		Metadata:           fmt.Sprintf(`{"days_until_expiry": %d, "contract_type": "AMC"}`, actualDaysRemaining),
	}

	if err := s.notifRepo.Create(notification); err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to create notification: %v", err)
	} else {
		log.Printf("[CONTRACT_EXPIRY] Created AMC expiry notification for contract %s, %d days remaining", contract.PONumber, actualDaysRemaining)
	}

	// Send email notification
	if s.mailer != nil && customer.User.Email != "" {
		customerName := customer.User.Name
		if customerName == "" {
			customerName = customer.User.Email
		}

		dashboardURL := s.dashboardBaseURL
		emailBody := utils.AMCExpiryEmailTemplate(
			customerName,
			solutionName,
			contract.PONumber,
			expiryDate,
			strconv.Itoa(actualDaysRemaining),
			dashboardURL,
		)

		subject := fmt.Sprintf("⚠️ AMC Expiry Notice - %d Days Remaining", actualDaysRemaining)
		if err := s.mailer.Send(customer.User.Email, subject, emailBody); err != nil {
			log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to send email to %s: %v", customer.User.Email, err)
		} else {
			log.Printf("[CONTRACT_EXPIRY] Sent AMC expiry email to %s for contract %s", customer.User.Email, contract.PONumber)
		}
	}
}

func (s *ContractExpiryService) sendWarrantyExpiryNotification(contract models.CustomerSolution, daysUntilExpiry int) {
	// Get customer details
	customer, err := s.customerRepo.GetByID(contract.CustomerID)
	if err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to get customer %s: %v", contract.CustomerID, err)
		return
	}

	// Determine notification type
	var notifType models.NotificationType
	switch daysUntilExpiry {
	case 90:
		notifType = models.NotificationTypeWarrantyExpiry3Months
	case 30:
		notifType = models.NotificationTypeWarrantyExpiry1Month
	case 7:
		notifType = models.NotificationTypeWarrantyExpiry7Days
	}

	solutionName := "Unknown Solution"
	if contract.Solution.Title != "" {
		solutionName = contract.Solution.Title
	}

	expiryDate := ""
	if contract.WarrantyEndDate != nil {
		expiryDate = contract.WarrantyEndDate.Format("02 Jan 2006")
	}

	// Calculate actual days remaining for display
	actualDaysRemaining := 0
	if contract.WarrantyEndDate != nil {
		actualDaysRemaining = int(time.Until(*contract.WarrantyEndDate).Hours() / 24)
		if actualDaysRemaining < 0 {
			actualDaysRemaining = 0
		}
	}

	title := fmt.Sprintf("Warranty Expiring Soon - %d Days Remaining", actualDaysRemaining)
	message := fmt.Sprintf("Your warranty for %s (PO: %s) expires on %s. Consider upgrading to an AMC plan.",
		solutionName, contract.PONumber, expiryDate)

	// Create in-app notification
	notification := &models.Notification{
		ID:                 uuid.New(),
		UserID:             customer.UserID,
		CustomerSolutionID: &contract.ID,
		Type:               notifType,
		Title:              title,
		Message:            message,
		IsRead:             false,
		Metadata:           fmt.Sprintf(`{"days_until_expiry": %d, "contract_type": "Warranty"}`, actualDaysRemaining),
	}

	if err := s.notifRepo.Create(notification); err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to create notification: %v", err)
	} else {
		log.Printf("[CONTRACT_EXPIRY] Created warranty expiry notification for contract %s, %d days remaining", contract.PONumber, actualDaysRemaining)
	}

	// Send email notification
	if s.mailer != nil && customer.User.Email != "" {
		customerName := customer.User.Name
		if customerName == "" {
			customerName = customer.User.Email
		}

		dashboardURL := s.dashboardBaseURL
		emailBody := utils.WarrantyExpiryEmailTemplate(
			customerName,
			solutionName,
			contract.PONumber,
			expiryDate,
			strconv.Itoa(actualDaysRemaining),
			dashboardURL,
		)

		subject := fmt.Sprintf("🛡️ Warranty Expiry Notice - %d Days Remaining", actualDaysRemaining)
		if err := s.mailer.Send(customer.User.Email, subject, emailBody); err != nil {
			log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to send email to %s: %v", customer.User.Email, err)
		} else {
			log.Printf("[CONTRACT_EXPIRY] Sent warranty expiry email to %s for contract %s", customer.User.Email, contract.PONumber)
		}
	}
}

// hasNotificationBeenSent checks if a notification has already been sent for this contract and type
func (s *ContractExpiryService) hasNotificationBeenSent(contractID uuid.UUID, notifType models.NotificationType) bool {
	// Use the repository method to check if notification was ever sent for this contract and type
	sent, err := s.notifRepo.HasContractNotificationBeenSent(contractID, notifType)
	if err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to check notification status: %v", err)
		return false // Proceed to send notification if check fails
	}
	return sent
}

/* =========================
   GET CONTRACTS FOR ADMIN VIEW
========================= */

type ContractWithCustomer struct {
	models.CustomerSolution
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	CompanyName   string `json:"company_name"`
	DaysRemaining int    `json:"days_remaining"`
	Status        string `json:"status"` // active, expiring_soon, expired
}

func (s *ContractExpiryService) GetAllAMCContractsWithDetails() ([]ContractWithCustomer, error) {
	contracts, err := s.customerSolRepo.GetAllAMCContracts()
	if err != nil {
		return nil, err
	}

	return s.enrichContractsWithCustomerInfo(contracts, "AMC")
}

func (s *ContractExpiryService) GetAllWarrantyContractsWithDetails() ([]ContractWithCustomer, error) {
	contracts, err := s.customerSolRepo.GetAllWarrantyContracts()
	if err != nil {
		return nil, err
	}

	return s.enrichContractsWithCustomerInfo(contracts, "Warranty")
}

func (s *ContractExpiryService) enrichContractsWithCustomerInfo(contracts []models.CustomerSolution, contractType string) ([]ContractWithCustomer, error) {
	var result []ContractWithCustomer

	for _, contract := range contracts {
		customer, err := s.customerRepo.GetByID(contract.CustomerID)
		if err != nil {
			continue
		}

		customerName := customer.User.Name
		customerEmail := customer.User.Email
		companyName := customer.Company.Name

		// Calculate days remaining
		var endDate *time.Time
		if contractType == "AMC" {
			endDate = contract.AMCEndDate
		} else {
			endDate = contract.WarrantyEndDate
		}

		daysRemaining := 0
		status := "active"

		if endDate != nil {
			daysRemaining = int(time.Until(*endDate).Hours() / 24)

			if daysRemaining < 0 {
				status = "expired"
				daysRemaining = 0
			} else if daysRemaining <= 30 {
				status = "expiring_soon"
			}
		}

		result = append(result, ContractWithCustomer{
			CustomerSolution: contract,
			CustomerName:     customerName,
			CustomerEmail:    customerEmail,
			CompanyName:      companyName,
			DaysRemaining:    daysRemaining,
			Status:           status,
		})
	}

	return result, nil
}

/* =========================
   ADMIN DAILY SUMMARY NOTIFICATION - DISABLED
   Daily admin notifications have been removed.
   Only customers receive 3 milestone notifications: 3 months, 1 month, 7 days before expiry
========================= */

func (s *ContractExpiryService) sendAdminDailySummary(amcNotificationsSent, warrantyNotificationsSent int) {
	// Disabled - Daily admin notifications removed per requirements
	// Admins can view contracts in admin dashboard anytime
}

// Legacy function - kept for compatibility but not used
func (s *ContractExpiryService) sendAdminDailySummaryLegacy(amcNotificationsSent, warrantyNotificationsSent int) {
	// Get counts of contracts expiring within different periods for the summary
	amcExpiring7Days, _ := s.customerSolRepo.GetExpiringAMCs(7)
	amcExpiring30Days, _ := s.customerSolRepo.GetExpiringAMCs(30)
	warrantyExpiring7Days, _ := s.customerSolRepo.GetExpiringWarranties(7)
	warrantyExpiring30Days, _ := s.customerSolRepo.GetExpiringWarranties(30)

	totalExpiringSoon := len(amcExpiring7Days) + len(warrantyExpiring7Days)
	totalExpiringMonth := len(amcExpiring30Days) + len(warrantyExpiring30Days)

	// Only send admin notification if there are contracts expiring or notifications were sent
	if totalExpiringSoon == 0 && totalExpiringMonth == 0 && amcNotificationsSent == 0 && warrantyNotificationsSent == 0 {
		log.Println("[CONTRACT_EXPIRY] No expiring contracts found, skipping admin notification")
		return
	}

	// Get all admin users
	admins, err := s.authRepo.GetUsersByRole(models.RoleAdmin)
	if err != nil {
		log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to get admin users: %v", err)
		return
	}

	if len(admins) == 0 {
		log.Println("[CONTRACT_EXPIRY] No admin users found")
		return
	}

	// Check if we already sent admin notification today
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.AddDate(0, 0, 1)

	// Create title and message
	title := "📋 Daily Contract Expiry Alert"
	message := fmt.Sprintf(
		"Today's Summary:\n• %d notifications sent today (%d AMC, %d Warranty)\n• %d contracts expiring within 7 days\n• %d contracts expiring within 30 days\n\nPlease review the AMC & Warranty pages for details.",
		amcNotificationsSent+warrantyNotificationsSent,
		amcNotificationsSent,
		warrantyNotificationsSent,
		totalExpiringSoon,
		totalExpiringMonth,
	)

	// Send notification to each admin
	for _, admin := range admins {
		// Check if notification already sent to this admin today
		var existingCount int64
		s.db.Model(&models.Notification{}).
			Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?",
				admin.ID, models.NotificationTypeAdminExpiryAlert, today, tomorrow).
			Count(&existingCount)

		if existingCount > 0 {
			continue // Already sent today
		}

		notification := &models.Notification{
			ID:       uuid.New(),
			UserID:   admin.ID,
			Type:     models.NotificationTypeAdminExpiryAlert,
			Title:    title,
			Message:  message,
			IsRead:   false,
			Metadata: fmt.Sprintf(`{"amc_sent": %d, "warranty_sent": %d, "expiring_7_days": %d, "expiring_30_days": %d}`, amcNotificationsSent, warrantyNotificationsSent, totalExpiringSoon, totalExpiringMonth),
		}

		if err := s.notifRepo.Create(notification); err != nil {
			log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to create admin notification for %s: %v", admin.Email, err)
		} else {
			log.Printf("[CONTRACT_EXPIRY] Sent daily summary notification to admin: %s", admin.Email)
		}

		// Also send email to admin
		if s.mailer != nil && admin.Email != "" {
			emailBody := s.getAdminSummaryEmailTemplate(
				admin.Name,
				amcNotificationsSent,
				warrantyNotificationsSent,
				len(amcExpiring7Days),
				len(warrantyExpiring7Days),
				len(amcExpiring30Days),
				len(warrantyExpiring30Days),
			)

			subject := fmt.Sprintf("📋 Daily Contract Expiry Alert - %s", time.Now().Format("02 Jan 2006"))
			if err := s.mailer.Send(admin.Email, subject, emailBody); err != nil {
				log.Printf("[CONTRACT_EXPIRY_ERROR] Failed to send admin email to %s: %v", admin.Email, err)
			} else {
				log.Printf("[CONTRACT_EXPIRY] Sent daily summary email to admin: %s", admin.Email)
			}
		}
	}
}

func (s *ContractExpiryService) getAdminSummaryEmailTemplate(
	adminName string,
	amcNotificationsSent int,
	warrantyNotificationsSent int,
	amcExpiring7Days int,
	warrantyExpiring7Days int,
	amcExpiring30Days int,
	warrantyExpiring30Days int,
) string {
	dashboardURL := s.dashboardBaseURL + "/admin/contracts/amc"

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 24px;">📋 Daily Contract Expiry Alert</h1>
        <p style="color: rgba(255,255,255,0.9); margin: 10px 0 0 0;">%s</p>
    </div>
    
    <div style="background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px;">
        <p style="font-size: 16px;">Hello %s,</p>
        
        <p>Here's your daily summary of contract expiry notifications:</p>
        
        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #667eea;">
            <h3 style="margin: 0 0 15px 0; color: #667eea;">📧 Notifications Sent Today</h3>
            <table style="width: 100%%; border-collapse: collapse;">
                <tr>
                    <td style="padding: 8px 0;">AMC Expiry Notifications:</td>
                    <td style="padding: 8px 0; text-align: right; font-weight: bold;">%d</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0;">Warranty Expiry Notifications:</td>
                    <td style="padding: 8px 0; text-align: right; font-weight: bold;">%d</td>
                </tr>
            </table>
        </div>
        
        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #f59e0b;">
            <h3 style="margin: 0 0 15px 0; color: #f59e0b;">⚠️ Contracts Expiring Within 7 Days</h3>
            <table style="width: 100%%; border-collapse: collapse;">
                <tr>
                    <td style="padding: 8px 0;">AMC Contracts:</td>
                    <td style="padding: 8px 0; text-align: right; font-weight: bold; color: %s;">%d</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0;">Warranty Contracts:</td>
                    <td style="padding: 8px 0; text-align: right; font-weight: bold; color: %s;">%d</td>
                </tr>
            </table>
        </div>
        
        <div style="background: white; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #3b82f6;">
            <h3 style="margin: 0 0 15px 0; color: #3b82f6;">📅 Contracts Expiring Within 30 Days</h3>
            <table style="width: 100%%; border-collapse: collapse;">
                <tr>
                    <td style="padding: 8px 0;">AMC Contracts:</td>
                    <td style="padding: 8px 0; text-align: right; font-weight: bold;">%d</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0;">Warranty Contracts:</td>
                    <td style="padding: 8px 0; text-align: right; font-weight: bold;">%d</td>
                </tr>
            </table>
        </div>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="display: inline-block; padding: 14px 30px; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; text-decoration: none; border-radius: 8px; font-weight: bold;">View Contract Details</a>
        </div>
        
        <p style="color: #666; font-size: 14px; margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
            This is an automated daily summary. You will receive this notification every day when there are expiring contracts or notifications to review.
        </p>
    </div>
</body>
</html>
`, time.Now().Format("Monday, January 02, 2006"),
		adminName,
		amcNotificationsSent,
		warrantyNotificationsSent,
		getColorForCount(amcExpiring7Days), amcExpiring7Days,
		getColorForCount(warrantyExpiring7Days), warrantyExpiring7Days,
		amcExpiring30Days,
		warrantyExpiring30Days,
		dashboardURL,
	)
}

func getColorForCount(count int) string {
	if count > 0 {
		return "#ef4444" // red
	}
	return "#22c55e" // green
}
