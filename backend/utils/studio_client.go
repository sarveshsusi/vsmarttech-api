package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// StudioClient talks to WhatsApp Studio public API (/api/v1).
type StudioClient struct {
	baseURL      string
	apiKey       string
	templateLang string
	tmplCreated  string
	tmplStatus   string
	tmplClosed   string
	httpClient   *http.Client

	// in-memory retry queue (process-local durable enough for Lightsail restarts
	// via re-notify on next ticket event; production can swap for DB later).
	mu    sync.Mutex
	queue []pendingSend
}

type pendingSend struct {
	idempotencyKey string
	to             string
	templateName   string
	customerName   string
	bodyParams     []string
	attempts       int
	nextAt         time.Time
}

type StudioConfig struct {
	BaseURL         string
	APIKey          string
	TemplateLang    string
	TemplateCreated string
	TemplateStatus  string
	TemplateClosed  string
}

func NewStudioClient(cfg StudioConfig) *StudioClient {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	key := strings.TrimSpace(cfg.APIKey)
	if base == "" || key == "" {
		return nil
	}
	lang := strings.TrimSpace(cfg.TemplateLang)
	if lang == "" {
		lang = "en_US"
	}
	c := &StudioClient{
		baseURL:      base,
		apiKey:       key,
		templateLang: lang,
		tmplCreated:  strings.TrimSpace(cfg.TemplateCreated),
		tmplStatus:   strings.TrimSpace(cfg.TemplateStatus),
		tmplClosed:   strings.TrimSpace(cfg.TemplateClosed),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
	go c.retryLoop()
	return c
}

type studioMessageRequest struct {
	To           string   `json:"to"`
	Type         string   `json:"type"`
	TemplateName string   `json:"template_name,omitempty"`
	Language     string   `json:"language,omitempty"`
	CustomerName string   `json:"customer_name,omitempty"`
	BodyParams   []string `json:"body_params,omitempty"`
}

type studioMessageResponse struct {
	Data struct {
		MessageID         string `json:"message_id"`
		WhatsAppMessageID string `json:"whatsapp_message_id"`
		ConversationID    string `json:"conversation_id"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// FormatPhoneForStudio normalizes CRM phones toward E.164 (not India-only).
func FormatPhoneForStudio(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "+") {
		digits := make([]rune, 0, len(raw))
		for _, r := range raw[1:] {
			if r >= '0' && r <= '9' {
				digits = append(digits, r)
			}
		}
		if len(digits) >= 8 && len(digits) <= 15 {
			return "+" + string(digits)
		}
		return ""
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
	if len(s) == 11 && strings.HasPrefix(s, "0") {
		return FormatPhoneForStudio(s[1:])
	}
	if len(s) == 12 && strings.HasPrefix(s, "91") {
		return "+" + s
	}
	if len(s) >= 8 && len(s) <= 15 {
		return "+" + s
	}
	return ""
}

func (c *StudioClient) sendTemplate(idempotencyKey, to, templateName, customerName string, bodyParams []string) error {
	if c == nil || templateName == "" || to == "" {
		return fmt.Errorf("studio: missing client/template/to")
	}
	payload := studioMessageRequest{
		To:           to,
		Type:         "template",
		TemplateName: templateName,
		Language:     c.templateLang,
		CustomerName: customerName,
		BodyParams:   bodyParams,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
	if res.StatusCode == 409 {
		var parsed studioMessageResponse
		_ = json.Unmarshal(raw, &parsed)
		if parsed.Error != nil && parsed.Error.Code == "opted_out" {
			log.Printf("[STUDIO_WA] skipped opted_out to=%s", to)
			return nil
		}
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("studio status=%s body=%s", res.Status, string(raw))
	}
	var parsed studioMessageResponse
	if err := json.Unmarshal(raw, &parsed); err == nil && parsed.Data.WhatsAppMessageID != "" {
		log.Printf("[STUDIO_WA] ok wamid=%s msg=%s", parsed.Data.WhatsAppMessageID, parsed.Data.MessageID)
	}
	return nil
}

func (c *StudioClient) enqueueOrSend(idemKey, to, templateName, customerName string, bodyParams []string) {
	if err := c.sendTemplate(idemKey, to, templateName, customerName, bodyParams); err != nil {
		log.Printf("[STUDIO_WA] send failed key=%s: %v — queued for retry", idemKey, err)
		c.mu.Lock()
		c.queue = append(c.queue, pendingSend{
			idempotencyKey: idemKey,
			to:             to,
			templateName:   templateName,
			customerName:   customerName,
			bodyParams:     bodyParams,
			attempts:       1,
			nextAt:         time.Now().Add(30 * time.Second),
		})
		c.mu.Unlock()
		return
	}
}

func (c *StudioClient) retryLoop() {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for range t.C {
		c.mu.Lock()
		var keep []pendingSend
		now := time.Now()
		for _, p := range c.queue {
			if p.nextAt.After(now) {
				keep = append(keep, p)
				continue
			}
			if p.attempts >= 5 {
				log.Printf("[STUDIO_WA] giving up key=%s after %d attempts", p.idempotencyKey, p.attempts)
				continue
			}
			err := c.sendTemplate(p.idempotencyKey, p.to, p.templateName, p.customerName, p.bodyParams)
			if err != nil {
				p.attempts++
				delay := time.Duration(p.attempts*p.attempts) * 30 * time.Second
				p.nextAt = now.Add(delay)
				keep = append(keep, p)
				log.Printf("[STUDIO_WA] retry scheduled key=%s attempt=%d", p.idempotencyKey, p.attempts)
			} else {
				log.Printf("[STUDIO_WA] retry ok key=%s", p.idempotencyKey)
			}
		}
		c.queue = keep
		c.mu.Unlock()
	}
}

func (c *StudioClient) notifyAsync(event, phone, customerName, ticketID, templateName string, bodyParams []string) {
	if c == nil {
		return
	}
	if templateName == "" {
		log.Printf("[STUDIO_WA] skip %s ticket=%s: template env empty", event, ticketID)
		return
	}
	to := FormatPhoneForStudio(phone)
	if to == "" {
		log.Printf("[STUDIO_WA] skip %s ticket=%s: invalid phone", event, ticketID)
		return
	}
	name := strings.TrimSpace(customerName)
	if name == "" {
		name = "Customer"
	}
	idem := fmt.Sprintf("ticket:%s:%s", ticketID, event)
	go c.enqueueOrSend(idem, to, templateName, name, bodyParams)
}

// NotifyTicketCreatedWhatsApp sends the created template with retries + idempotency.
func (c *StudioClient) NotifyTicketCreatedWhatsApp(phone, customerName, ticketID, title string) {
	c.notifyAsync("created", phone, customerName, ticketID, c.tmplCreated, []string{customerNameOr(customerName), ticketID, title})
}

// NotifyTicketStatusWhatsApp sends the status-changed template.
func (c *StudioClient) NotifyTicketStatusWhatsApp(phone, customerName, ticketID, status, title string) {
	c.notifyAsync("status:"+status, phone, customerName, ticketID, c.tmplStatus, []string{customerNameOr(customerName), ticketID, status, title})
}

// NotifyTicketClosedWhatsApp sends the closed template.
func (c *StudioClient) NotifyTicketClosedWhatsApp(phone, customerName, ticketID, title string) {
	c.notifyAsync("closed", phone, customerName, ticketID, c.tmplClosed, []string{customerNameOr(customerName), ticketID, title})
}

func customerNameOr(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "Customer"
	}
	return n
}
