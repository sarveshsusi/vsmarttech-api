package utils

import (
	"fmt"
	"html"
	"strings"
)

// Shared Vsmart email design system.
// Uses nested tables + inline styles so messages render correctly in Outlook,
// Gmail, Apple Mail, and mobile clients.

func escape(s string) string {
	return html.EscapeString(s)
}

func emailHeadStyles() string {
	// Progressive enhancement only — critical layout/colors are inlined below.
	return `
body,table,td,a{-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%}
table,td{mso-table-lspace:0pt;mso-table-rspace:0pt}
img{-ms-interpolation-mode:bicubic;border:0;outline:none;text-decoration:none}
body{margin:0!important;padding:0!important;width:100%!important;background:#EEF2F7}
a{color:#1D4ED8}
@media only screen and (max-width:620px){
  .email-container{width:100%!important}
  .stack-pad{padding-left:20px!important;padding-right:20px!important}
}
`
}

func renderEmail(title, brandEyebrow, heading, subheading, bodyHTML string) string {
	title = escape(title)
	brandEyebrow = escape(brandEyebrow)
	heading = escape(heading)
	subheading = escape(subheading)

	subBlock := ""
	if subheading != "" {
		subBlock = fmt.Sprintf(
			`<tr><td style="padding:0 0 4px 0;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:22px;color:#64748B;">%s</td></tr>`,
			subheading,
		)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml" xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:v="urn:schemas-microsoft-com:vml">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="x-apple-disable-message-reformatting">
<meta name="format-detection" content="telephone=no,address=no,email=no,date=no,url=no">
<title>%s</title>
<!--[if mso]>
<noscript>
<xml>
  <o:OfficeDocumentSettings>
    <o:AllowPNG/>
    <o:PixelsPerInch>96</o:PixelsPerInch>
  </o:OfficeDocumentSettings>
</xml>
</noscript>
<style type="text/css">
  table{border-collapse:collapse}
  td{font-family:Arial,Helvetica,sans-serif}
</style>
<![endif]-->
<style type="text/css">%s</style>
</head>
<body style="margin:0;padding:0;background:#EEF2F7;width:100%%;">
  <div style="display:none;max-height:0;overflow:hidden;mso-hide:all;">%s</div>
  <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%" style="background:#EEF2F7;margin:0;padding:0;width:100%%;">
    <tr>
      <td align="center" style="padding:28px 12px;">
        <!--[if mso]>
        <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="600"><tr><td>
        <![endif]-->
        <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%" class="email-container" style="max-width:600px;width:100%%;background:#FFFFFF;border:1px solid #E2E8F0;">
          <tr>
            <td bgcolor="#0B1F3A" style="background:#0B1F3A;padding:18px 28px;font-family:Arial,Helvetica,sans-serif;font-size:12px;letter-spacing:1px;text-transform:uppercase;font-weight:700;color:#FFFFFF;">
              Vsmart Technologies · %s
            </td>
          </tr>
          <tr>
            <td class="stack-pad" style="padding:28px 28px 8px 28px;">
              <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%">
                <tr>
                  <td style="padding:0 0 8px 0;font-family:Arial,Helvetica,sans-serif;font-size:24px;line-height:32px;font-weight:700;color:#0F172A;">
                    %s
                  </td>
                </tr>
                %s
              </table>
            </td>
          </tr>
          <tr>
            <td class="stack-pad" style="padding:8px 28px 28px 28px;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:24px;color:#334155;">
              %s
            </td>
          </tr>
          <tr>
            <td bgcolor="#F8FAFC" style="background:#F8FAFC;border-top:1px solid #E2E8F0;padding:18px 28px 24px 28px;text-align:center;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:18px;color:#94A3B8;">
              © 2026 Vsmart Technologies. All rights reserved.<br>
              This is an automated message from the Vsmart CRM. Please do not reply to this email.
            </td>
          </tr>
        </table>
        <!--[if mso]>
        </td></tr></table>
        <![endif]-->
      </td>
    </tr>
  </table>
</body>
</html>`, title, emailHeadStyles(), heading, brandEyebrow, heading, subBlock, bodyHTML)
}

func paragraph(text string) string {
	return fmt.Sprintf(
		`<p style="margin:0 0 16px 0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:24px;color:#334155;">%s</p>`,
		text,
	)
}

func ctaButton(href, label string) string {
	safeHref := escape(href)
	safeLabel := escape(label)
	if strings.TrimSpace(href) == "" {
		return `<p style="margin:16px 0 0 0;font-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:20px;color:#B45309;">Open the Vsmart CRM portal to continue.</p>`
	}
	return fmt.Sprintf(`
<table role="presentation" cellpadding="0" cellspacing="0" border="0" align="center" style="margin:24px auto 8px auto;">
  <tr>
    <td bgcolor="#1D4ED8" style="background:#1D4ED8;border-radius:8px;mso-padding-alt:14px 28px;">
      <a href="%s" target="_blank" rel="noopener noreferrer" style="display:inline-block;padding:14px 28px;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:18px;font-weight:700;color:#FFFFFF;text-decoration:none;">%s</a>
    </td>
  </tr>
</table>
<p style="margin:8px 0 0 0;text-align:center;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:18px;color:#64748B;word-break:break-all;">
  Or open: <a href="%s" style="color:#1D4ED8;text-decoration:underline;">%s</a>
</p>
`, safeHref, safeLabel, safeHref, safeHref)
}

func detailPanel(rows [][2]string) string {
	var b strings.Builder
	b.WriteString(`<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="margin:16px 0;background:#F8FAFC;border:1px solid #E2E8F0;">`)
	for i, row := range rows {
		border := "border-bottom:1px solid #E2E8F0;"
		if i == len(rows)-1 {
			border = ""
		}
		b.WriteString(fmt.Sprintf(`
<tr>
  <td style="padding:10px 14px;%sfont-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:20px;color:#64748B;font-weight:700;width:34%%;vertical-align:top;">%s</td>
  <td style="padding:10px 14px;%sfont-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:20px;color:#0F172A;font-weight:600;vertical-align:top;">%s</td>
</tr>`, border, escape(row[0]), border, escape(row[1])))
	}
	b.WriteString(`</table>`)
	return b.String()
}

func badge(text, bg, fg string) string {
	return fmt.Sprintf(
		`<span style="display:inline-block;padding:4px 10px;background:%s;color:%s;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:16px;font-weight:700;border-radius:999px;">%s</span>`,
		bg, fg, escape(text),
	)
}

func callout(borderColor, bg, fg, htmlInner string) string {
	return fmt.Sprintf(`
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%" style="margin:16px 0;background:%s;border-left:4px solid %s;">
  <tr>
    <td style="padding:14px 16px;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:22px;color:%s;">
      %s
    </td>
  </tr>
</table>`, bg, borderColor, fg, htmlInner)
}

func bigNumber(value, caption, color string) string {
	return fmt.Sprintf(`
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%" style="margin:16px 0;background:#F8FAFC;border:1px solid #E2E8F0;">
  <tr>
    <td align="center" style="padding:20px 16px;">
      <div style="font-family:Arial,Helvetica,sans-serif;font-size:44px;line-height:48px;font-weight:700;color:%s;">%s</div>
      <div style="font-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:18px;color:#64748B;margin-top:4px;">%s</div>
    </td>
  </tr>
</table>`, color, escape(value), escape(caption))
}

// SetPasswordEmailTemplate — account activation / set password
func SetPasswordEmailTemplate(userName, resetURL string) string {
	body := paragraph("Hello "+escape(userName)+",") +
		paragraph("Your Vsmart CRM account has been created. Set a password to activate access and start using the portal.") +
		ctaButton(resetURL, "Set password") +
		callout("#2563EB", "#EFF6FF", "#1E3A8A", "<strong>Security:</strong> This link expires in 24 hours. If you did not expect this email, contact your administrator.")
	return renderEmail("Set Your Password", "Account setup", "Welcome to Vsmart", "Activate your account to continue", body)
}

// PasswordResetEmailTemplate — forgot password
func PasswordResetEmailTemplate(resetURL string) string {
	body := paragraph("We received a request to reset your Vsmart CRM password.") +
		ctaButton(resetURL, "Reset password") +
		callout("#D97706", "#FFFBEB", "#92400E", "<strong>Expires in 24 hours.</strong> If you did not request a reset, you can safely ignore this email.")
	return renderEmail("Reset Your Password", "Security", "Reset your password", "Use the secure link below", body)
}

// OTPEmailTemplate — login OTP
func OTPEmailTemplate(code string) string {
	body := paragraph("Use this one-time code to finish signing in to Vsmart CRM.") +
		fmt.Sprintf(`
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%" style="margin:16px 0;background:#F8FAFC;border:1px solid #E2E8F0;">
  <tr>
    <td align="center" style="padding:22px 16px;">
      <div style="font-family:Consolas,Monaco,monospace;font-size:36px;line-height:44px;letter-spacing:8px;font-weight:700;color:#1D4ED8;">%s</div>
      <div style="margin-top:8px;font-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:18px;color:#64748B;">Expires in 5 minutes</div>
    </td>
  </tr>
</table>`, escape(code)) +
		callout("#2563EB", "#EFF6FF", "#1E3A8A", "<strong>Do not share this code</strong> with anyone. Vsmart staff will never ask for it.")
	return renderEmail("Your Login Code", "Verification", "Your login code", "Enter this code to continue", body)
}

// TicketCreatedEmailTemplate — customer opened a ticket
func TicketCreatedEmailTemplate(customerName, ticketNumber, ticketTitle, dashboardURL string) string {
	body := paragraph("Hello "+escape(customerName)+",") +
		paragraph("We have received your support request. Our team will review it and assign an engineer shortly.") +
		detailPanel([][2]string{
			{"Ticket ID", ticketNumber},
			{"Title", ticketTitle},
			{"Status", "Open"},
		}) +
		`<div style="margin:8px 0 0 0;">` + badge("OPEN", "#DBEAFE", "#1D4ED8") + `</div>` +
		ctaButton(dashboardURL, "View ticket")
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
	body := paragraph("Hello "+escape(customerName)+",") +
		paragraph("Your support ticket has been closed. If anything still needs attention, reply via the portal or open a new request.") +
		detailPanel([][2]string{
			{"Ticket ID", ticketNumber},
			{"Title", ticketTitle},
			{"Resolved by", engineerName},
			{"Closed on", closureDate},
		}) +
		`<div style="margin:8px 0 0 0;">` + badge("CLOSED", "#D1FAE5", "#047857") + `</div>` +
		callout("#059669", "#ECFDF5", "#065F46", "<strong>Resolution note</strong><br>"+escape(comment)) +
		ctaButton(dashboardURL, "View ticket details")
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
	body := callout("#DC2626", "#FEF2F2", "#7F1D1D", "<strong>Action required:</strong> A ticket has been escalated and needs immediate attention.") +
		detailPanel([][2]string{
			{"Ticket ID", ticketID},
			{"Title", title},
			{"Status", status},
			{"Duration", "Open more than 7 days"},
		}) +
		ctaButton(dashboardURL, "Open ticket")
	return renderEmail("Ticket Escalation", "Alert", "Ticket escalation alert", "Immediate review needed", body)
}

func contractUrgency(daysRemaining string) (label, bg, fg string) {
	switch daysRemaining {
	case "7":
		return "Urgent", "#FEE2E2", "#B91C1C"
	case "30":
		return "Important", "#FEF3C7", "#B45309"
	default:
		return "Reminder", "#DBEAFE", "#1D4ED8"
	}
}

// AMCExpiryEmailTemplate — AMC expiry notice
func AMCExpiryEmailTemplate(customerName, solutionName, poNumber, expiryDate, daysRemaining, dashboardURL string) string {
	urgencyText, bg, fg := contractUrgency(daysRemaining)
	body := paragraph("Dear "+escape(customerName)+",") +
		paragraph("Your Annual Maintenance Contract (AMC) is approaching expiry. Renew soon to avoid service interruption.") +
		`<div style="margin:0 0 8px 0;">` + badge(urgencyText, bg, fg) + `</div>` +
		bigNumber(daysRemaining, "Days remaining", "#1D4ED8") +
		detailPanel([][2]string{
			{"Solution", solutionName},
			{"PO Number", poNumber},
			{"Expiry date", expiryDate},
		}) +
		ctaButton(dashboardURL, "View in dashboard")
	return renderEmail("AMC Expiry Notice", "Contracts", "AMC expiry notice", "Renewal reminder", body)
}

// WarrantyExpiryEmailTemplate — warranty expiry notice
func WarrantyExpiryEmailTemplate(customerName, solutionName, poNumber, expiryDate, daysRemaining, dashboardURL string) string {
	urgencyText, bg, fg := contractUrgency(daysRemaining)
	body := paragraph("Dear "+escape(customerName)+",") +
		paragraph("Your product warranty is approaching expiry. Contact support to discuss renewal or an AMC plan.") +
		`<div style="margin:0 0 8px 0;">` + badge(urgencyText, bg, fg) + `</div>` +
		bigNumber(daysRemaining, "Days remaining", "#1D4ED8") +
		detailPanel([][2]string{
			{"Solution", solutionName},
			{"PO Number", poNumber},
			{"Expiry date", expiryDate},
		}) +
		callout("#2563EB", "#EFF6FF", "#1E3A8A", "<strong>Tip:</strong> Extending coverage or moving to AMC keeps support uninterrupted.") +
		ctaButton(dashboardURL, "View in dashboard")
	return renderEmail("Warranty Expiry Notice", "Contracts", "Warranty expiry notice", "Renewal reminder", body)
}

// SLABreachEmailTemplate — SLA breach alert
func SLABreachEmailTemplate(recipientName, ticketNumber, customerName, ticketTitle, priority string, hoursOverdue, slaHours int, dashboardURL string) string {
	body := paragraph("Hello "+escape(recipientName)+",") +
		paragraph("A support ticket has exceeded its SLA deadline and needs immediate attention.") +
		callout("#DC2626", "#FEF2F2", "#7F1D1D", fmt.Sprintf("<strong>SLA breached</strong> — overdue by %d hours.", hoursOverdue)) +
		detailPanel([][2]string{
			{"Ticket ID", ticketNumber},
			{"Title", ticketTitle},
			{"Customer", customerName},
			{"Priority", priority},
			{"SLA", fmt.Sprintf("%d hours", slaHours)},
			{"Overdue by", fmt.Sprintf("%d hours", hoursOverdue)},
		}) +
		ctaButton(dashboardURL, "View ticket details") +
		callout("#D97706", "#FFFBEB", "#92400E", "<strong>Next steps:</strong> Review the ticket, update status with a comment, and escalate if resolution will take longer.")
	return renderEmail("SLA Breach Alert", "SLA", "SLA breach alert", "Immediate action required", body)
}
