package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"

	"rbac/models"
)

//go:embed migrations/*.sql
var gooseMigrations embed.FS

// Migrate applies schema changes according to mode:
//   - "auto": GORM AutoMigrate (local/dev)
//   - "goose": versioned SQL via goose (production default)
//
// For goose mode on a brand-new database (no users table), AutoMigrate runs once
// to create the baseline schema, then goose applies forward migrations.
func Migrate(db *gorm.DB, mode string) {
	switch mode {
	case "goose":
		if !db.Migrator().HasTable(&models.User{}) {
			log.Println("goose mode: empty database — applying AutoMigrate baseline once")
			autoMigrate(db)
		}
		if err := runGoose(db); err != nil {
			log.Fatalf("goose migration failed: %v", err)
		}
		syncEngineerIDs(db)
		log.Println("Database migration completed (goose)")
	default: // auto
		// GORM cannot ADD ... NOT NULL over existing rows; backfill first.
		ensureRefreshTokenFamilyIDColumn(db)
		autoMigrate(db)
		backfillRefreshTokenFamilyIDs(db)
		backfillTicketFeedbackLifecycle(db)
		syncEngineerIDs(db)
		log.Println("Database migration completed (auto)")
	}
}

// backfillTicketFeedbackLifecycle migrates legacy ticket_feedbacks rows for AutoMigrate mode.
func backfillTicketFeedbackLifecycle(db *gorm.DB) {
	if !db.Migrator().HasTable("ticket_feedbacks") {
		return
	}
	_ = db.Exec(`ALTER TABLE ticket_feedbacks ALTER COLUMN rating DROP NOT NULL`).Error
	_ = db.Exec(`
		UPDATE ticket_feedbacks
		SET remarks = LEFT(COALESCE(NULLIF(TRIM(comment), ''), remarks), 500)
		WHERE EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'ticket_feedbacks' AND column_name = 'comment'
		)
		AND (remarks IS NULL OR remarks = '')
		AND comment IS NOT NULL AND TRIM(comment) <> ''
	`).Error
	_ = db.Exec(`
		UPDATE ticket_feedbacks
		SET
			feedback_status = 'Submitted',
			submitted_at = COALESCE(submitted_at, created_at),
			updated_at = COALESCE(updated_at, created_at, NOW())
		WHERE rating IS NOT NULL AND rating >= 1
		  AND (feedback_status IS NULL OR feedback_status = '' OR feedback_status = 'Pending')
		  AND submitted_at IS NULL
	`).Error
	_ = db.Exec(`
		UPDATE ticket_feedbacks tf
		SET
			customer_id = t.customer_id,
			company_id = c.company_id
		FROM tickets t
		JOIN customers c ON c.id = t.customer_id
		WHERE tf.ticket_id = t.id
		  AND (tf.customer_id IS NULL OR tf.company_id IS NULL
		       OR tf.customer_id = '00000000-0000-0000-0000-000000000000'
		       OR tf.company_id = '00000000-0000-0000-0000-000000000000')
	`).Error
	_ = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_ticket_feedbacks_ticket_id_unique ON ticket_feedbacks (ticket_id)`).Error
}

// ensureRefreshTokenFamilyIDColumn adds family_id as nullable, backfills from id,
// then sets NOT NULL — so AutoMigrate does not fail on populated refresh_tokens.
func ensureRefreshTokenFamilyIDColumn(db *gorm.DB) {
	if !db.Migrator().HasTable("refresh_tokens") {
		return
	}
	if err := db.Exec(`ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS family_id UUID`).Error; err != nil {
		log.Printf("warning: add refresh_tokens.family_id: %v", err)
		return
	}
	backfillRefreshTokenFamilyIDs(db)
	if err := db.Exec(`ALTER TABLE refresh_tokens ALTER COLUMN family_id SET NOT NULL`).Error; err != nil {
		log.Printf("warning: set refresh_tokens.family_id NOT NULL: %v", err)
	}
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_id ON refresh_tokens (family_id)`).Error; err != nil {
		log.Printf("warning: index refresh_tokens.family_id: %v", err)
	}
}

func autoMigrate(db *gorm.DB) {
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
		&models.Asset{},
		&models.AssetStatusHistory{},

		/* =========================
		   TICKETING SYSTEM
		========================= */
		&models.Ticket{},
		&models.TicketAssignment{},
		&models.TicketStatusHistory{},
		&models.TicketEvent{},
		&models.TicketComment{},
		&models.TicketAttachment{},
		&models.TicketFeedback{},

		/* =========================
		   FIELD SERVICE & AUDIT
		========================= */
		&models.ServiceVisit{},
		&models.ServiceVisitProof{},
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
		log.Fatalf("Database AutoMigrate failed: %v", err)
	}
}

func backfillRefreshTokenFamilyIDs(db *gorm.DB) {
	res := db.Exec(`
		UPDATE refresh_tokens
		SET family_id = id
		WHERE family_id IS NULL
		   OR family_id = '00000000-0000-0000-0000-000000000000'
	`)
	if res.Error != nil {
		log.Printf("warning: refresh_tokens family_id backfill: %v", res.Error)
	}
}

func runGoose(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	return runGooseSQL(sqlDB)
}

func runGooseSQL(sqlDB *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	goose.SetBaseFS(gooseMigrations)
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return err
	}
	return nil
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
		log.Printf("warning: failed to sync engineer IDs: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("Synced engineer_id for %d tickets", result.RowsAffected)
	}
}
