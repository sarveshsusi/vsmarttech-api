package database

import (
	"log"

	"rbac/models"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(

		/* =========================
		   AUTH & SECURITY
		========================= */
		&models.User{},
		&models.RefreshToken{},
		&models.PasswordResetToken{},
		&models.TwoFAOTP{},
		&models.RememberedDevice{},

		/* =========================
		   CORE ACTORS
		========================= */
		&models.Customer{},
		&models.SupportEngineer{},

		/* =========================
		   SOLUTION & CONTRACT
		========================= */
		&models.Solution{},
		&models.CustomerSolution{}, // PO + AMC/Warranty lives here

		/* =========================
		   TICKETING SYSTEM
		========================= */
		&models.Ticket{},
		&models.TicketAssignment{},
		&models.TicketStatusHistory{},
		&models.TicketComment{},
		&models.TicketAttachment{},
		&models.TicketFeedback{},

		/* =========================
		   FIELD SERVICE & AUDIT
		========================= */
		&models.ServiceVisit{},
		&models.GPSLog{},
		&models.DigitalSignature{},
		&models.AuditLog{},

		/* =========================
		   ESCALATION
		========================= */
		&models.EscalationRule{},
		&models.TicketEscalation{},
		&models.Company{},

		/* =========================
		   AMC SCHEDULER
		========================= */
		&models.AMCAssignment{},
		&models.AMCVisit{},
		&models.AMCVisitProof{},

		/* =========================
		   NOTIFICATIONS
		========================= */
		&models.Notification{},
		&models.WebhookEvent{},
		&models.NotificationPreference{},
	)

	if err != nil {
		log.Fatalf("❌ Database migration failed: %v", err)
	}

	log.Println("✅ Database migration completed successfully")

	// Post-migration: Sync engineer_id from ticket_assignments to tickets
	syncEngineerIDs(db)
}

// syncEngineerIDs ensures tickets have engineer_id set from ticket_assignments
func syncEngineerIDs(db *gorm.DB) {
	result := db.Exec(`
		UPDATE tickets t
		SET engineer_id = ta.engineer_id
		FROM ticket_assignments ta
		WHERE t.id = ta.ticket_id
		AND t.engineer_id IS NULL
		AND ta.engineer_id IS NOT NULL
	`)

	if result.Error != nil {
		log.Printf("⚠️ Failed to sync engineer IDs: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("✅ Synced engineer_id for %d tickets", result.RowsAffected)
	}
}
