package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rbac/config"
	"rbac/models"
	"rbac/utils"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret: "test-secret",
			AccessExpiry: time.Minute,
		},
	}

	user := &models.User{
		ID:    uuid.New(),
		Email: "admin@example.com",
		Role:  models.RoleAdmin,
	}
	token, err := utils.GenerateAccessToken(user, cfg.JWT.AccessSecret, cfg.JWT.AccessExpiry)
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	t.Run("missing header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
		AuthMiddleware(cfg)(c)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", w.Code)
		}
	})

	t.Run("invalid bearer", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
		c.Request.Header.Set("Authorization", "Bearer not-a-token")
		AuthMiddleware(cfg)(c)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d", w.Code)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthMiddleware(cfg))
		r.GET("/x", func(c *gin.Context) {
			id, _ := c.Get(CtxUserID)
			c.JSON(http.StatusOK, gin.H{"user_id": id})
		})
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(CtxUserRole, models.RoleSupport)
		c.Next()
	})
	r.GET("/admin-only", RequireRole(models.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d", w.Code)
	}
}
