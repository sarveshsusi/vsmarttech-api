package database

import (
	"errors"
	"time"

	"rbac/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *config.Config) error {
	if cfg.Database.URL == "" {
		return errors.New("DATABASE_URL is required for Supabase")
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Connection pool (Supabase-friendly)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	DB = db
	return nil
}
