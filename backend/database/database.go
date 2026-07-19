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
		return errors.New("DATABASE_URL is required")
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

	maxOpen := cfg.Database.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.Database.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 3
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	life := cfg.Database.ConnMaxLife
	if life <= 0 {
		life = 30 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(life)

	DB = db
	return nil
}
