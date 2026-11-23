package service_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/handler"
	modtour "rbac/internal/modules/touradvance"
	"rbac/middleware"
	"rbac/models"
	"rbac/service"
)

func TestTourAdvanceRejectHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := service.OpenTourDBForTest(t)
	fx := service.SeedTourForTest(t, db)
	svc := service.NewTourAdvanceService(db, nil)
	created, err := svc.Create(fx.Engineer.ID, models.RoleSupport, service.InstallInputForTest(fx.SiteA.ID))
	if err != nil {
		t.Fatal(err)
	}

	h := handler.NewTourAdvanceHandler(svc)
	router := gin.New()
	api := router.Group("/api/v1")
	protected := api.Group("")
	protected.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.MustParse(c.GetHeader("X-User-Id")))
		c.Set(middleware.CtxUserRole, models.Role(c.GetHeader("X-User-Role")))
		c.Next()
	})
	admin := protected.Group("/admin")
	admin.Use(middleware.RequireRole(models.RoleAdmin, models.RoleSuperAdmin))
	modtour.RegisterAdmin(admin, h)
	adminOnly := protected.Group("/admin")
	adminOnly.Use(middleware.RequireRole(models.RoleAdmin))
	modtour.RegisterProcess(adminOnly, h)

	post := func(path, userID, role, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-User-Role", role)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	rejectPath := "/api/v1/admin/tour-advances/" + created.ID.String() + "/reject"

	adminRes := post(rejectPath, fx.Admin.ID.String(), string(models.RoleAdmin), `{"rejection_remarks":"no"}`)
	if adminRes.Code != http.StatusForbidden {
		t.Fatalf("admin status=%d body=%s", adminRes.Code, adminRes.Body.String())
	}

	supportRes := post(rejectPath, fx.Engineer.ID.String(), string(models.RoleSupport), `{"rejection_remarks":"no"}`)
	if supportRes.Code != http.StatusForbidden {
		t.Fatalf("support status=%d body=%s", supportRes.Code, supportRes.Body.String())
	}

	missing := post("/api/v1/admin/tour-advances/"+uuid.NewString()+"/reject", fx.SuperAdmin.ID.String(), string(models.RoleSuperAdmin), `{"rejection_remarks":"gone"}`)
	if missing.Code != http.StatusNotFound || !bytes.Contains(missing.Body.Bytes(), []byte("request not found")) {
		t.Fatalf("missing status=%d body=%s", missing.Code, missing.Body.String())
	}

	ok := post(rejectPath, fx.SuperAdmin.ID.String(), string(models.RoleSuperAdmin), `{"rejection_remarks":"PO information needs correction"}`)
	if ok.Code != http.StatusOK {
		t.Fatalf("reject status=%d body=%s", ok.Code, ok.Body.String())
	}
	var view map[string]any
	if err := json.Unmarshal(ok.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view["status"] != string(models.TourStatusRejected) {
		t.Fatalf("status %#v", view["status"])
	}
	if view["rejection_remarks"] != "PO information needs correction" {
		t.Fatalf("remarks %#v", view["rejection_remarks"])
	}
	events, _ := view["events"].([]any)
	foundEvent := false
	for _, item := range events {
		event, _ := item.(map[string]any)
		if event["action"] == "rejected" && event["note"] == "PO information needs correction" {
			foundEvent = true
		}
	}
	if !foundEvent {
		t.Fatalf("events %#v", view["events"])
	}

	var stored models.TourAdvanceRequest
	if err := db.First(&stored, "id = ?", created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.TourStatusRejected {
		t.Fatalf("stored status %s", stored.Status)
	}

	again := post(rejectPath, fx.SuperAdmin.ID.String(), string(models.RoleSuperAdmin), `{"rejection_remarks":"again"}`)
	if again.Code != http.StatusConflict || !bytes.Contains(again.Body.Bytes(), []byte("invalid state transition")) {
		t.Fatalf("second reject status=%d body=%s", again.Code, again.Body.String())
	}

	approve := post("/api/v1/admin/tour-advances/"+created.ID.String()+"/approve", fx.SuperAdmin.ID.String(), string(models.RoleSuperAdmin), `{"approved_amount":"1000"}`)
	if approve.Code != http.StatusConflict || !bytes.Contains(approve.Body.Bytes(), []byte("invalid state transition")) {
		t.Fatalf("approve rejected status=%d body=%s", approve.Code, approve.Body.String())
	}

	bill := post("/api/v1/admin/tour-advances/"+created.ID.String()+"/expenses", fx.Admin.ID.String(), string(models.RoleAdmin),
		`{"expenses":[{"category":"FOOD","bill_type":"BILL","expense_date":"2026-10-02","description":"Meals","claimed_amount":"500","reference_number":"INV-1"}]}`)
	if bill.Code != http.StatusConflict || !bytes.Contains(bill.Body.Bytes(), []byte("invalid state transition")) {
		t.Fatalf("bill rejected status=%d body=%s", bill.Code, bill.Body.String())
	}
}
