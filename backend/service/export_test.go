package service

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"rbac/models"
)

func OpenTourDBForTest(t *testing.T) *gorm.DB {
	t.Helper()
	return openTourDB(t)
}

type TourHTTPFixture struct {
	Engineer   models.User
	Admin      models.User
	SuperAdmin models.User
	SiteA      models.Customer
}

func SeedTourForTest(t *testing.T, db *gorm.DB) TourHTTPFixture {
	t.Helper()
	fx := seedTour(t, db)
	return TourHTTPFixture{
		Engineer: fx.engineer, Admin: fx.admin, SuperAdmin: fx.superAdmin, SiteA: fx.siteA,
	}
}

func InstallInputForTest(siteID uuid.UUID) CreateTourAdvanceInput {
	return installInput(siteID)
}
