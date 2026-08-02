package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// StudioClient talks to WhatsApp Studio public API (/api/v1).
type StudioClient struct {
	baseURL       string
	apiKey        string
	templateLang  string
	tmplCreated   string
	tmplStatus    string
	tmplClosed    string
	httpClient    *http.Client
}

type StudioConfig struct {
	BaseURL              string
	APIKey               string
	TemplateLang         string
	TemplateCreated      string
	TemplateStatus       string
	TemplateClosed       string
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
	return &StudioClient{
		baseURL:      base,
		apiKey:       key,
		templateLang: lang,
		tmplCreated:  strings.TrimSpace(cfg.TemplateCreated),
		tmplStatus:   strings.TrimSpace(cfg.TemplateStatus),
		tmplClosed:   strings.TrimSpace(cfg.TemplateClosed),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

type studioMessageRequest struct {
	To           string   `json:"to"`
	Type         string   `json:"type"`
	TemplateName string   `json:"template_name,omitempty"`
	Language     string   `json:"language,omitempty"`
	CustomerName string   `json:"customer_name,omitempty"`
	BodyParams   []string `json:"body_params,omitempty"`
}

// FormatPhoneForStudio normalizes CRM phones toward E.164.
func FormatPhoneForStudio(raw string) string {
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

func (c *StudioClient) sendTemplate(to, templateName, customerName string, bodyParams []string) error {
	if c == nil || templateName == "" || to == "" {
		return nil
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

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode >= 300 {
		return fmt.Errorf("studio status=%s body=%s", res.Status, string(raw))
	}
	return nil
}

// NotifyTicketCreatedWhatsApp sends the created template (fire-and-forget).
func (c *StudioClient) NotifyTicketCreatedWhatsApp(phone, customerName, ticketID, title string) {
	if c == nil {
		return
	}
	to := FormatPhoneForStudio(phone)
	if to == "" || c.tmplCreated == "" {
		return
	}
	name := strings.TrimSpace(customerName)
	if name == "" {
		name = "Customer"
	}
	go func() {
		if err := c.sendTemplate(to, c.tmplCreated, name, []string{name, ticketID, title}); err != nil {
			log.Printf("[STUDIO_WA] ticket.created failed ticket=%s: %v", ticketID, err)
			return
		}
		log.Printf("[STUDIO_WA] ticket.created ok ticket=%s", ticketID)
	}()
}

// NotifyTicketStatusWhatsApp sends the status-changed template.
func (c *StudioClient) NotifyTicketStatusWhatsApp(phone, customerName, ticketID, status, title string) {
	if c == nil {
		return
	}
	to := FormatPhoneForStudio(phone)
	if to == "" || c.tmplStatus == "" {
		return
	}
	name := strings.TrimSpace(customerName)
	if name == "" {
		name = "Customer"
	}
	go func() {
		if err := c.sendTemplate(to, c.tmplStatus, name, []string{name, ticketID, status, title}); err != nil {
			log.Printf("[STUDIO_WA] ticket.status failed ticket=%s: %v", ticketID, err)
			return
		}
		log.Printf("[STUDIO_WA] ticket.status ok ticket=%s", ticketID)
	}()
}

// NotifyTicketClosedWhatsApp sends the closed template.
func (c *StudioClient) NotifyTicketClosedWhatsApp(phone, customerName, ticketID, title string) {
	if c == nil {
		return
	}
	to := FormatPhoneForStudio(phone)
	if to == "" || c.tmplClosed == "" {
		return
	}
	name := strings.TrimSpace(customerName)
	if name == "" {
		name = "Customer"
	}
	go func() {
		if err := c.sendTemplate(to, c.tmplClosed, name, []string{name, ticketID, title}); err != nil {
			log.Printf("[STUDIO_WA] ticket.closed failed ticket=%s: %v", ticketID, err)
			return
		}
		log.Printf("[STUDIO_WA] ticket.closed ok ticket=%s", ticketID)
	}()
}
