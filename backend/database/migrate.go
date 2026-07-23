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
		syncEngineerIDs(db)
		log.Println("Database migration completed (auto)")
	}
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
