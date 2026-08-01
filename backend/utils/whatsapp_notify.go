package utils

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// WhatsAppNotifyPayload is posted to Vsmart WhatsApp Studio
// POST /api/integrations/crm/notify
type WhatsAppNotifyPayload struct {
	Event         string `json:"event"`
	TicketID      string `json:"ticket_id"`
	Status        string `json:"status,omitempty"`
	Title         string `json:"title,omitempty"`
	CustomerPhone string `json:"customer_phone"`
	CustomerName  string `json:"customer_name,omitempty"`
}

// NotifyCustomerWhatsApp sends a fire-and-forget ticket event to WhatsApp Studio.
// No-op when WA_STUDIO_NOTIFY_URL / WA_STUDIO_SHARED_SECRET are unset or phone is empty.
func NotifyCustomerWhatsApp(payload WhatsAppNotifyPayload) {
	url := strings.TrimSpace(os.Getenv("WA_STUDIO_NOTIFY_URL"))
	secret := strings.TrimSpace(os.Getenv("WA_STUDIO_SHARED_SECRET"))
	phone := strings.TrimSpace(payload.CustomerPhone)
	if url == "" || secret == "" || phone == "" {
		return
	}
	if payload.Event == "" || payload.TicketID == "" {
		return
	}

	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[WA_NOTIFY] marshal error: %v", err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			log.Printf("[WA_NOTIFY] request build error: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+secret)

		client := &http.Client{Timeout: 15 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			log.Printf("[WA_NOTIFY] send error ticket=%s: %v", payload.TicketID, err)
			return
		}
		defer res.Body.Close()
		if res.StatusCode >= 300 {
			log.Printf(
				"[WA_NOTIFY] non-2xx ticket=%s status=%s",
				payload.TicketID,
				res.Status,
			)
			return
		}
		log.Printf("[WA_NOTIFY] ok event=%s ticket=%s", payload.Event, payload.TicketID)
	}()
}

// FormatCustomerPhoneForWhatsApp normalizes a CRM phone for E.164-ish delivery.
// If digits-only Indian 10-digit mobile, prefix +91.
func FormatCustomerPhoneForWhatsApp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "+") {
		return raw
	}
	digits := make([]rune, 0, len(raw))
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	s := string(digits)
	if len(s) == 10 && (s[0] == '6' || s[0] == '7' || s[0] == '8' || s[0] == '9') {
		return "+91" + s
	}
	if len(s) >= 8 {
		return "+" + s
	}
	return ""
}
