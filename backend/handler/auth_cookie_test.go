package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"rbac/config"
)

func TestRefreshCookiePartitionedOnlyForCrossSite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cross := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(cross)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	handler := &AuthHandler{cfg: &config.Config{
		Server: config.ServerConfig{Env: "production", CookieSameSite: "none"},
		JWT:    config.JWTConfig{RefreshExpiry: 7 * 24 * time.Hour},
	}}
	handler.setRefreshCookie(ctx, "refresh-value")

	headers := cross.Header().Values("Set-Cookie")
	joined := strings.Join(headers, "\n")
	if !strings.Contains(joined, "Partitioned") || !strings.Contains(joined, "HttpOnly") || !strings.Contains(joined, "Secure") || !strings.Contains(joined, "SameSite=None") {
		t.Fatalf("cross-site refresh cookie = %q", joined)
	}
	if !strings.Contains(joined, "Max-Age=604800") {
		t.Fatalf("refresh lifetime = %q", joined)
	}

	local := httptest.NewRecorder()
	localCtx, _ := gin.CreateTestContext(local)
	localCtx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	handler.cfg.Server.Env = "development"
	handler.cfg.Server.CookieSameSite = "lax"
	handler.setRefreshCookie(localCtx, "refresh-value")
	localHeader := strings.Join(local.Header().Values("Set-Cookie"), "\n")
	if strings.Contains(localHeader, "Partitioned") || strings.Contains(localHeader, "Secure") {
		t.Fatalf("local refresh cookie = %q", localHeader)
	}
	if !strings.Contains(localHeader, "HttpOnly") || !strings.Contains(localHeader, "SameSite=Lax") {
		t.Fatalf("local refresh cookie = %q", localHeader)
	}
}

func TestClearRefreshCookieExpiresPartitionedAndLegacy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	handler := &AuthHandler{cfg: &config.Config{
		Server: config.ServerConfig{Env: "production", CookieSameSite: "none"},
		JWT:    config.JWTConfig{RefreshExpiry: time.Hour},
	}}
	handler.clearRefreshCookie(ctx)
	headers := w.Header().Values("Set-Cookie")
	if len(headers) != 2 {
		t.Fatalf("clear cookies = %v", headers)
	}
	var partitioned, legacy bool
	for _, header := range headers {
		if !strings.Contains(header, "Max-Age=0") && !strings.Contains(header, "refresh_token=;") {
			t.Fatalf("cookie not cleared: %s", header)
		}
		if strings.Contains(header, "Partitioned") {
			partitioned = true
		} else {
			legacy = true
		}
	}
	if !partitioned || !legacy {
		t.Fatalf("missing clear variants: %v", headers)
	}
}
