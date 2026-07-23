package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseIsReplacementQuery(t *testing.T) {
	tests := []struct {
		raw     string
		want    *bool
		wantErr bool
	}{
		{raw: "", want: nil},
		{raw: "true", want: boolPtr(true)},
		{raw: "1", want: boolPtr(true)},
		{raw: "yes", want: boolPtr(true)},
		{raw: "false", want: boolPtr(false)},
		{raw: "0", want: boolPtr(false)},
		{raw: "no", want: boolPtr(false)},
		{raw: "maybe", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseIsReplacementQuery(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("raw=%q: expected error", tt.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("raw=%q: unexpected error: %v", tt.raw, err)
		}
		if tt.want == nil {
			if got != nil {
				t.Fatalf("raw=%q: got %v want nil", tt.raw, *got)
			}
			continue
		}
		if got == nil || *got != *tt.want {
			t.Fatalf("raw=%q: got %v want %v", tt.raw, got, *tt.want)
		}
	}
}

func TestListRejectsInvalidIsReplacement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &AssetHandler{}
	r := gin.New()
	r.GET("/assets", h.List)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets?is_replacement=maybe", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func boolPtr(v bool) *bool { return &v }
