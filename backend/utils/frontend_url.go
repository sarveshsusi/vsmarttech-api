package utils

import (
	"net/url"
	"strings"
)

// JoinFrontendURL builds an absolute app URL from FRONTEND_URL and a path
// (path may include a query string). Trailing slashes on the base are trimmed.
// Returns "" if base is empty so callers can fail closed instead of linking to localhost.
func JoinFrontendURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

// CustomerTicketDetailURL deep-links a customer/support ticket detail page.
func CustomerTicketDetailURL(base, ticketID string) string {
	id := strings.TrimSpace(ticketID)
	if id == "" {
		return JoinFrontendURL(base, "/customer/tickets")
	}
	return JoinFrontendURL(base, "/tickets/"+url.PathEscape(id))
}

// AdminTicketDetailURL deep-links the admin ticket details page.
func AdminTicketDetailURL(base, ticketID string) string {
	id := strings.TrimSpace(ticketID)
	if id == "" {
		return JoinFrontendURL(base, "/admin/tickets")
	}
	return JoinFrontendURL(base, "/admin/tickets/details?id="+url.QueryEscape(id))
}

// SupportReportsURL is the support SLA / reports page.
func SupportReportsURL(base string) string {
	return JoinFrontendURL(base, "/support/reports")
}

// ResetPasswordURL builds the set/reset password link.
func ResetPasswordURL(base, token string) string {
	return JoinFrontendURL(base, "/reset-password?token="+url.QueryEscape(token))
}
