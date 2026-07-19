package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(RequestIDMiddleware())
	r.GET("/x", func(c *gin.Context) {
		id, _ := c.Get(CtxRequestID)
		c.String(http.StatusOK, "%v", id)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", "fixed-id-123")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("X-Request-ID"); got != "fixed-id-123" {
		t.Fatalf("header=%q", got)
	}
	if w.Body.String() != "fixed-id-123" {
		t.Fatalf("body=%q", w.Body.String())
	}
}
