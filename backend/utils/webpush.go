package utils

import (
	"encoding/json"
	"log"

	"rbac/config"
	"rbac/models"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// PushPayload is the JSON body delivered to the service worker.
type PushPayload struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	URL            string `json:"url"`
	NotificationID string `json:"notification_id,omitempty"`
}

// WebPusher sends Web Push messages. Nil-safe when VAPID is not configured.
type WebPusher struct {
	publicKey  string
	privateKey string
	subject    string
}

func NewWebPusher(cfg config.VAPIDConfig) *WebPusher {
	if cfg.PublicKey == "" || cfg.PrivateKey == "" {
		return nil
	}
	subject := cfg.Subject
	if subject == "" {
		subject = "mailto:admin@localhost"
	}
	return &WebPusher{
		publicKey:  cfg.PublicKey,
		privateKey: cfg.PrivateKey,
		subject:    subject,
	}
}

func (w *WebPusher) PublicKey() string {
	if w == nil {
		return ""
	}
	return w.publicKey
}

func (w *WebPusher) Enabled() bool {
	return w != nil && w.publicKey != "" && w.privateKey != ""
}

// Send delivers a push to one subscription. Returns HTTP status from the push service.
// Callers should delete the subscription when status is 404 or 410.
func (w *WebPusher) Send(sub *models.PushSubscription, payload PushPayload) (int, error) {
	if !w.Enabled() || sub == nil {
		return 0, nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	resp, err := webpush.SendNotification(body, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}, &webpush.Options{
		Subscriber:      w.subject,
		VAPIDPublicKey:  w.publicKey,
		VAPIDPrivateKey: w.privateKey,
		TTL:             60 * 60 * 24,
	})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("[WEBPUSH] push service status=%d endpoint=%s", resp.StatusCode, truncateEndpoint(sub.Endpoint))
	}
	return resp.StatusCode, nil
}

func truncateEndpoint(endpoint string) string {
	if len(endpoint) <= 64 {
		return endpoint
	}
	return endpoint[:64] + "…"
}
