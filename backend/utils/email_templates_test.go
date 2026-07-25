package utils

import (
	"strings"
	"testing"
)

func TestJoinFrontendURL(t *testing.T) {
	got := JoinFrontendURL("https://crm.vsmarttec.net/", "/tickets/VS%2F07%2F26%2F1")
	want := "https://crm.vsmarttec.net/tickets/VS%2F07%2F26%2F1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if JoinFrontendURL("", "/tickets") != "" {
		t.Fatal("empty base must not invent a localhost URL")
	}
}

func TestCustomerTicketDetailURLEncodesSlash(t *testing.T) {
	got := CustomerTicketDetailURL("https://crm.vsmarttec.net", "VS/07/26/1")
	if !strings.Contains(got, "/tickets/VS%2F07%2F26%2F1") {
		t.Fatalf("expected path-escaped ticket id, got %q", got)
	}
	if strings.Contains(got, "localhost") {
		t.Fatal("must not contain localhost")
	}
}

func TestResetPasswordURLUsesFrontendOrigin(t *testing.T) {
	got := ResetPasswordURL("https://crm.vsmarttec.net/", "abc+123")
	if !strings.HasPrefix(got, "https://crm.vsmarttec.net/reset-password?token=") {
		t.Fatalf("unexpected reset URL: %q", got)
	}
	if !strings.Contains(got, "abc%2B123") {
		t.Fatalf("expected query-escaped token, got %q", got)
	}
}

func TestTicketCreatedEmailTemplateContainsKeyContent(t *testing.T) {
	html := TicketCreatedEmailTemplate(
		"Jane",
		"VS/07/26/1",
		"Gate fault",
		"https://crm.example.com/tickets/VS%2F07%2F26%2F1",
	)
	for _, want := range []string{
		"Jane",
		"VS/07/26/1",
		"Gate fault",
		"Vsmart Technologies",
		"https://crm.example.com/tickets/VS%2F07%2F26%2F1",
		`role="presentation"`,
		`bgcolor="#1D4ED8"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected email HTML to contain %q", want)
		}
	}
	if strings.Contains(html, "localhost") {
		t.Fatal("email must not contain localhost")
	}
}

func TestTicketClosureEmailTemplateEscapesAndStyles(t *testing.T) {
	html := TicketClosureEmailTemplate(
		"Jane",
		"VS/07/26/1",
		"Gate fault",
		"Boobalan",
		"21 Jul 2026",
		`<script>alert(1)</script>`,
		"https://crm.example.com/tickets/VS%2F07%2F26%2F1",
	)
	if !strings.Contains(html, "CLOSED") {
		t.Fatal("expected closed badge")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("expected closure comment to be HTML-escaped")
	}
	if strings.Contains(html, "<script>alert") {
		t.Fatal("raw script must not appear in email HTML")
	}
	if !strings.Contains(html, `bgcolor="#1D4ED8"`) {
		t.Fatal("expected Outlook-safe CTA button")
	}
	if !strings.Contains(html, `role="presentation"`) {
		t.Fatal("expected table-based layout for Outlook")
	}
}

func TestSLABreachEmailTemplateUsesProvidedURL(t *testing.T) {
	html := SLABreachEmailTemplate(
		"Admin",
		"VS/07/26/1",
		"Acme",
		"Printer down",
		"High",
		4,
		24,
		"https://crm.vsmarttec.net/admin/tickets/details?id=VS%2F07%2F26%2F1",
	)
	if !strings.Contains(html, "https://crm.vsmarttec.net/admin/tickets/details?id=VS%2F07%2F26%2F1") {
		t.Fatal("expected production deep link in SLA email")
	}
	if strings.Contains(html, "localhost") {
		t.Fatal("SLA email must not contain localhost")
	}
}
