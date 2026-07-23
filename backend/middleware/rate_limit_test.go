package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestRateLimitKeyPrefersUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "10.0.0.1:1234"

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	c.Set(CtxUserID, id)

	got := rateLimitKey(c)
	want := "user:11111111-1111-1111-1111-111111111111"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRateLimitKeyFallsBackToIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = "10.0.0.9:9999"

	got := rateLimitKey(c)
	if got != "ip:10.0.0.9" {
		t.Fatalf("got %q", got)
	}
}

func TestRateLimitPerUserNotShared(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if u := c.GetHeader("X-User"); u != "" {
			c.Set(CtxUserID, u)
		}
		c.Next()
	})
	r.Use(RateLimit(2))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	do := func(user string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = "10.0.0.1:1"
		if user != "" {
			req.Header.Set("X-User", user)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	if do("a") != 200 || do("a") != 200 || do("a") != 429 {
		t.Fatal("user a should hit limit on 3rd request")
	}
	if do("b") != 200 {
		t.Fatal("user b should have a separate bucket")
	}
}
