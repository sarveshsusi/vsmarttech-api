package utils

import (
	"fmt"
	"html"
	"strings"
)

// Shared Vsmart email design system (table-based for client compatibility).

func escape(s string) string {
	return html.EscapeString(s)
}

func emailBaseStyles() string {
	return `
body{margin:0;padding:0;background:#EEF2F7;font-family:Arial,Helvetica,sans-serif;color:#0F172A;-webkit-font-smoothing:antialiased}
.wrapper{width:100%;background:#EEF2F7;padding:32px 12px}
.card{max-width:600px;margin:0 auto;background:#FFFFFF;border-radius:16px;overflow:hidden;border:1px solid #E2E8F0}
.brand{padding:18px 28px;background:#0B1F3A;color:#FFFFFF;font-size:13px;letter-spacing:0.08em;text-transform:uppercase;font-weight:700}
.hero{padding:28px 28px 12px 28px}
.hero h1{margin:0 0 8px 0;font-size:24px;line-height:1.3;color:#0F172A;font-weight:700}
.hero p{margin:0;font-size:14px;line-height:1.6;color:#64748B}
.body{padding:8px 28px 28px 28px}
.p{margin:0 0 16px 0;font-size:15px;line-height:1.65;color:#334155}
.muted{color:#64748B;font-size:13px;line-height:1.6}
.panel{background:#F8FAFC;border:1px solid #E2E8F0;border-radius:12px;padding:16px 18px;margin:18px 0}
.row{padding:8px 0;border-bottom:1px solid #E2E8F0;font-size:14px}
.row:last-child{border-bottom:none}
.label{color:#64748B;display:inline-block;min-width:110px;font-weight:600}
.value{color:#0F172A;font-weight:600}
.badge{display:inline-block;padding:4px 10px;border-radius:999px;font-size:12px;font-weight:700;letter-spacing:0.02em}
.badge-blue{background:#DBEAFE;color:#1D4ED8}
.badge-green{background:#D1FAE5;color:#047857}
.badge-red{background:#FEE2E2;color:#B91C1C}
.badge-amber{background:#FEF3C7;color:#B45309}
.callout{border-radius:12px;padding:14px 16px;margin:18px 0;font-size:14px;line-height:1.55}
.callout-blue{background:#EFF6FF;border-left:4px solid #2563EB;color:#1E3A8A}
.callout-green{background:#ECFDF5;border-left:4px solid #059669;color:#065F46}
.callout-red{background:#FEF2F2;border-left:4px solid #DC2626;color:#7F1D1D}
.callout-amber{background:#FFFBEB;border-left:4px solid #D97706;color:#92400E}
.cta-wrap{text-align:center;margin:28px 0 8px 0}
.cta{display:inline-block;background:#1D4ED8;color:#FFFFFF !important;text-decoration:none;padding:14px 28px;border-radius:10px;font-weight:700;font-size:14px}
.cta:hover{background:#1E40AF}
.otp{font-size:36px;letter-spacing:10px;font-weight:700;color:#1D4ED8;font-family:Consolas,Monaco,monospace;margin:8px 0}
.foot{padding:18px 28px 24px 28px;background:#F8FAFC;border-top:1px solid #E2E8F0;text-align:center;color:#94A3B8;font-size:12px;line-height:1.6}
.foot a{color:#64748B;text-decoration:underline}
.big-num{font-size:44px;font-weight:700;line-height:1;margin:4px 0}
`
}

func renderEmail(title, brandEyebrow, heading, subheading, bodyHTML string) string {
	title = escape(title)
	brandEyebrow = escape(brandEyebrow)
	heading = escape(heading)
	subheading = escape(subheading)

	subBlock := ""
	if subheading != "" {
		subBlock = fmt.Sprintf(`<p>%s</p>`, subheading)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<title>%s</title>
<style>%s</style>
</head>
<body>
  <div class="wrapper">
    <div class="card">
      <div class="brand">Vsmart Technologies · %s</div>
      <div class="hero">
        <h1>%s</h1>
        %s
      </div>
      <div class="body">
        %s
      </div>
      <div class="foot">
        <p>© 2026 Vsmart Technologies. All rights reserved.</p>
        <p>This is an automated message from the Vsmart CRM. Please do not reply to this email.</p>
      </div>
    </div>
  </div>
</body>
</html>`, title, emailBaseStyles(), brandEyebrow, heading, subBlock, bodyHTML)
}

func ctaButton(href, label string) string {
	return fmt.Sprintf(`
<div class="cta-wrap">
  <a class="cta" href="%s" target="_blank" rel="noopener noreferrer">%s</a>
</div>
<p class="muted" style="text-align:center;word-break:break-all;">Or open: <a href="%s">%s</a></p>
`, escape(href), escape(label), escape(href), escape(href))
}

func detailPanel(rows [][2]string) string {
	var b strings.Builder
	b.WriteString(`<div class="panel">`)
	for _, row := range rows {
		b.WriteString(fmt.Sprintf(
			`<div class="row"><span class="label">%s</span><span class="value">%s</span></div>`,
			escape(row[0]), escape(row[1]),
		))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// SetPasswordEmailTemplate — account activation / set password
func SetPasswordEmailTemplate(userName, resetURL string) string {
	body := fmt.Sprintf(`
<p class="p">Hello %s,</p>
<p class="p">Your Vsmart CRM account has been created. Set a password to activate access and start using the portal.</p>
%s
<div class="callout callout-blue"><strong>Security:</strong> This link expires in 24 hours. If you did not expect this email, contact your administrator.</div>
`, escape(userName), ctaButton(resetURL, "Set password"))
	return renderEmail("Set Your Password", "Account setup", "Welcome to Vsmart", "Activate your account to continue", body)
}

// PasswordResetEmailTemplate — forgot password
func PasswordResetEmailTemplate(resetURL string) string {
	body := fmt.Sprintf(`
<p class="p">We received a request to reset your Vsmart CRM password.</p>
%s
<div class="callout callout-amber"><strong>Expires in 24 hours.</strong> If you did not request a reset, you can safely ignore this email.</div>
`, ctaButton(resetURL, "Reset password"))
	return renderEmail("Reset Your Password", "Security", "Reset your password", "Use the secure link below", body)
}

// OTPEmailTemplate — login OTP
func OTPEmailTemplate(code string) string {
	body := fmt.Sprintf(`
<p class="p">Use this one-time code to finish signing in to Vsmart CRM.</p>
<div class="panel" style="text-align:center;">
  <div class="otp">%s</div>
  <p class="muted" style="margin:0;">Expires in 5 minutes</p>
</div>
<div class="callout callout-blue"><strong>Do not share this code</strong> with anyone. Vsmart staff will never ask for it.</div>
`, escape(code))
	return renderEmail("Your Login Code", "Verification", "Your login code", "Enter this code to continue", body)
}

// TicketCreatedEmailTemplate — customer opened a ticket
func TicketCreatedEmailTemplate(customerName, ticketNumber, ticketTitle, dashboardURL string) string {
	body := fmt.Sprintf(`
<p class="p">Hello %s,</p>
<p class="p">We have received your support request. Our team will review it and assign an engineer shortly.</p>
%s
<span class="badge badge-blue">OPEN</span>
%s
`, escape(customerName),
		detailPanel([][2]string{
			{"Ticket ID", ticketNumber},
			{"Title", ticketTitle},
			{"Status", "Open"},
		}),
		ctaButton(dashboardURL, "View ticket"),
	)
	return renderEmail(
		"Ticket Received",
		"Support",
		"Your ticket has been created",
		"We will keep you updated as it progresses",
		body,
	)
}

// TicketClosureEmailTemplate — ticket closed (customer notification)
func TicketClosureEmailTemplate(customerName, ticketNumber, ticketTitle, engineerName, closureDate, closureComment, dashboardURL string) string {
	comment := strings.TrimSpace(closureComment)
	if comment == "" {
		comment = "No additional comment was provided."
	}
	body := fmt.Sprintf(`
<p class="p">Hello %s,</p>
<p class="p">Your support ticket has been closed. If anything still needs attention, reply via the portal or open a new request.</p>
%s
<span class="badge badge-green">CLOSED</span>
<div class="callout callout-green"><strong>Resolution note</strong><br>%s</div>
%s
`, escape(customerName),
		detailPanel([][2]string{
			{"Ticket ID", ticketNumber},
			{"Title", ticketTitle},
			{"Resolved by", engineerName},
			{"Closed on", closureDate},
		}),
		escape(comment),
		ctaButton(dashboardURL, "View ticket details"),
	)
	return renderEmail(
		"Ticket Resolved",
		"Support",
		"Your ticket has been resolved",
		fmt.Sprintf("Ticket %s", escape(ticketNumber)),
		body,
	)
}

// TicketEscalationEmailTemplate — escalation alert
func TicketEscalationEmailTemplate(ticketID, title, status, dashboardURL string) string {
	body := fmt.Sprintf(`
<div class="callout callout-red"><strong>Action required:</strong> A ticket has been escalated and needs immediate attention.</div>
%s
%s
`, detailPanel([][2]string{
		{"Ticket ID", ticketID},
		{"Title", title},
		{"Status", status},
		{"Duration", "Open more than 7 days"},
	}), ctaButton(dashboardURL, "Open ticket"))
	return renderEmail("Ticket Escalation", "Alert", "Ticket escalation alert", "Immediate review needed", body)
}

// AMCExpiryEmailTemplate — AMC expiry notice
func AMCExpiryEmailTemplate(customerName, solutionName, poNumber, expiryDate, daysRemaining, dashboardURL string) string {
	urgencyText := "Reminder"
	badge := "badge-blue"
	if daysRemaining == "7" {
		urgencyText = "Urgent"
		badge = "badge-red"
	} else if daysRemaining == "30" {
		urgencyText = "Important"
		badge = "badge-amber"
	}
	body := fmt.Sprintf(`
<p class="p">Dear %s,</p>
<p class="p">Your Annual Maintenance Contract (AMC) is approaching expiry. Renew soon to avoid service interruption.</p>
<span class="badge %s">%s</span>
<div class="panel" style="text-align:center;">
  <div class="big-num" style="color:#1D4ED8;">%s</div>
  <div class="muted">Days remaining</div>
</div>
%s
%s
`, escape(customerName), badge, escape(urgencyText), escape(daysRemaining),
		detailPanel([][2]string{
			{"Solution", solutionName},
			{"PO Number", poNumber},
			{"Expiry date", expiryDate},
		}),
		ctaButton(dashboardURL, "View in dashboard"),
	)
	return renderEmail("AMC Expiry Notice", "Contracts", "AMC expiry notice", "Renewal reminder", body)
}

// WarrantyExpiryEmailTemplate — warranty expiry notice
func WarrantyExpiryEmailTemplate(customerName, solutionName, poNumber, expiryDate, daysRemaining, dashboardURL string) string {
	urgencyText := "Reminder"
	badge := "badge-blue"
	if daysRemaining == "7" {
		urgencyText = "Urgent"
		badge = "badge-red"
	} else if daysRemaining == "30" {
		urgencyText = "Important"
		badge = "badge-amber"
	}
	body := fmt.Sprintf(`
<p class="p">Dear %s,</p>
<p class="p">Your product warranty is approaching expiry. Contact support to discuss renewal or an AMC plan.</p>
<span class="badge %s">%s</span>
<div class="panel" style="text-align:center;">
  <div class="big-num" style="color:#1D4ED8;">%s</div>
  <div class="muted">Days remaining</div>
</div>
%s
<div class="callout callout-blue"><strong>Tip:</strong> Extending coverage or moving to AMC keeps support uninterrupted.</div>
%s
`, escape(customerName), badge, escape(urgencyText), escape(daysRemaining),
		detailPanel([][2]string{
			{"Solution", solutionName},
			{"PO Number", poNumber},
			{"Expiry date", expiryDate},
		}),
		ctaButton(dashboardURL, "View in dashboard"),
	)
	return renderEmail("Warranty Expiry Notice", "Contracts", "Warranty expiry notice", "Renewal reminder", body)
}

// SLABreachEmailTemplate — SLA breach alert
func SLABreachEmailTemplate(recipientName, ticketNumber, customerName, ticketTitle, priority string, hoursOverdue, slaHours int, dashboardURL string) string {
	body := fmt.Sprintf(`
<p class="p">Hello %s,</p>
<p class="p">A support ticket has exceeded its SLA deadline and needs immediate attention.</p>
<div class="callout callout-red"><strong>SLA breached</strong> — overdue by %d hours.</div>
%s
%s
<div class="callout callout-amber"><strong>Next steps:</strong> Review the ticket, update status with a comment, and escalate if resolution will take longer.</div>
`, escape(recipientName), hoursOverdue,
		detailPanel([][2]string{
			{"Ticket ID", ticketNumber},
			{"Title", ticketTitle},
			{"Customer", customerName},
			{"Priority", priority},
			{"SLA", fmt.Sprintf("%d hours", slaHours)},
			{"Overdue by", fmt.Sprintf("%d hours", hoursOverdue)},
		}),
		ctaButton(dashboardURL, "View ticket details"),
	)
	return renderEmail("SLA Breach Alert", "SLA", "SLA breach alert", "Immediate action required", body)
}
