package utils

import (
	"strings"
	"testing"
)

func TestTicketCreatedEmailTemplateContainsKeyContent(t *testing.T) {
	html := TicketCreatedEmailTemplate(
		"Jane",
		"VS/07/26/1",
		"Gate fault",
		"https://crm.example.com/customer/tickets",
	)
	for _, want := range []string{
		"Jane",
		"VS/07/26/1",
		"Gate fault",
		"Vsmart Technologies",
		"https://crm.example.com/customer/tickets",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected email HTML to contain %q", want)
		}
	}
}

func TestTicketClosureEmailTemplateEscapesAndStyles(t *testing.T) {
	html := TicketClosureEmailTemplate(
		"Jane",
		"VS/07/26/1",
		"Gate fault",
		"Boobalan",
		"21 Jul 2026",
		"Fixed on site",
		"https://crm.example.com/customer/tickets",
	)
	if !strings.Contains(html, "CLOSED") {
		t.Fatal("expected closed badge")
	}
	if !strings.Contains(html, "Fixed on site") {
		t.Fatal("expected closure comment")
	}
	if !strings.Contains(html, "class=\"cta\"") {
		t.Fatal("expected CTA button")
	}
}
