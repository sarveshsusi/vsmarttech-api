package utils

import "fmt"

// SetPasswordEmailTemplate - Professional template for password setup
func SetPasswordEmailTemplate(userName, resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Set Your Password</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 700;
        }
        .content {
            padding: 40px 30px;
        }
        .greeting {
            font-size: 16px;
            color: #333;
            margin-bottom: 20px;
        }
        .message {
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .button-container {
            text-align: center;
            margin: 30px 0;
        }
        .button {
            display: inline-block;
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 14px 32px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 15px;
        }
        .link-text {
            color: #666;
            font-size: 12px;
            margin-top: 15px;
            word-break: break-all;
        }
        .security-note {
            background-color: #e6f2ff;
            border-left: 4px solid #0066cc;
            padding: 15px;
            margin-top: 30px;
            border-radius: 4px;
            font-size: 13px;
            color: #666;
        }
        .footer {
            background-color: #f9f9f9;
            padding: 20px;
            text-align: center;
            color: #999;
            font-size: 12px;
            border-top: 1px solid #eee;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 Set Your Password</h1>
        </div>
        <div class="content">
            <div class="greeting">Hello %s,</div>
            <div class="message">Your account has been created successfully. Click the button below to set your password and activate your account.</div>
            <div class="button-container">
                <a href="%s" class="button">Set Password Now</a>
            </div>
            <div class="link-text">Or copy and paste: <a href="%s">%s</a></div>
            <div class="security-note">
                <strong>🔒 Security:</strong> This link expires in 24 hours.
            </div>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, userName, resetURL, resetURL, resetURL)
}

// PasswordResetEmailTemplate - Professional template for password reset
func PasswordResetEmailTemplate(resetURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your Password</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 700;
        }
        .content {
            padding: 40px 30px;
        }
        .message {
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .button {
            display: inline-block;
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 14px 32px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 15px;
        }
        .warning {
            background-color: #fff5f5;
            border-left: 4px solid #0066cc;
            padding: 15px;
            margin-top: 30px;
            border-radius: 4px;
            font-size: 13px;
            color: #666;
        }
        .footer {
            background-color: #f9f9f9;
            padding: 20px;
            text-align: center;
            color: #999;
            font-size: 12px;
            border-top: 1px solid #eee;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔑 Reset Your Password</h1>
        </div>
        <div class="content">
            <div class="message">We received a request to reset your password. Click below to create a new password.</div>
            <div style="text-align: center;">
                <a href="%s" class="button">Reset Password</a>
            </div>
            <div class="warning">
                <strong>⏰ Expires in 24 hours.</strong> If you didn't request this, you can safely ignore this email.
            </div>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, resetURL)
}

// OTPEmailTemplate - Professional template for OTP code
func OTPEmailTemplate(code string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Login Code</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 700;
        }
        .content {
            padding: 40px 30px;
        }
        .code-container {
            background: #e6f2ff;
            border: 2px solid #0066cc;
            border-radius: 8px;
            padding: 30px;
            text-align: center;
            margin: 30px 0;
        }
        .code {
            font-size: 48px;
            font-weight: 700;
            color: #0066cc;
            letter-spacing: 8px;
            margin: 0;
            font-family: monospace;
        }
        .instruction {
            background-color: #e6f2ff;
            border-left: 4px solid #0066cc;
            padding: 15px;
            margin-top: 30px;
            border-radius: 4px;
            font-size: 13px;
            color: #666;
        }
        .footer {
            background-color: #f9f9f9;
            padding: 20px;
            text-align: center;
            color: #999;
            font-size: 12px;
            border-top: 1px solid #eee;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔐 Your Login Code</h1>
        </div>
        <div class="content">
            <p>Your one-time security code for login:</p>
            <div class="code-container">
                <p class="code">%s</p>
                <p style="color: #f5576c; font-weight: 600;">⏰ Expires in 5 minutes</p>
            </div>
            <div class="instruction">
                <strong>📌 How to use:</strong> Enter this code in your login form. Do not share this code.
            </div>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, code)
}

// TicketEscalationEmailTemplate - Professional template for ticket escalation
func TicketEscalationEmailTemplate(ticketID, title, status, dashboardURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ticket Escalation Alert</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 700;
        }
        .content {
            padding: 40px 30px;
        }
        .alert-box {
            background-color: #e6f2ff;
            border-left: 4px solid #0066cc;
            padding: 20px;
            margin-bottom: 30px;
            border-radius: 4px;
        }
        .details {
            background-color: #f9f9f9;
            border: 1px solid #eee;
            border-radius: 6px;
            padding: 20px;
            margin: 20px 0;
        }
        .detail-row {
            margin-bottom: 10px;
        }
        .button {
            display: inline-block;
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 14px 32px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 15px;
        }
        .footer {
            background-color: #f9f9f9;
            padding: 20px;
            text-align: center;
            color: #999;
            font-size: 12px;
            border-top: 1px solid #eee;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚨 Ticket Escalation Alert</h1>
        </div>
        <div class="content">
            <div class="alert-box">
                <div style="color: #0066cc; font-weight: 700; font-size: 16px; margin-bottom: 10px;">High Priority: Ticket Escalation</div>
                <div>A ticket has been escalated - immediate action required!</div>
            </div>
            <div class="details">
                <div class="detail-row"><strong>📋 Ticket ID:</strong> %s</div>
                <div class="detail-row"><strong>📝 Title:</strong> %s</div>
                <div class="detail-row"><strong>📊 Status:</strong> %s</div>
                <div class="detail-row"><strong>⏱️ Duration:</strong> Open for more than 7 days</div>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s" class="button">View Ticket Details</a>
            </div>
            <p>Please review this ticket immediately and take action.</p>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, ticketID, title, status, dashboardURL)
}

// AMCExpiryEmailTemplate - Template for AMC expiry notifications
func AMCExpiryEmailTemplate(customerName, solutionName, poNumber, expiryDate, daysRemaining, dashboardURL string) string {
	urgencyColor := "#0066cc" // blue for warning
	urgencyText := "Reminder"
	if daysRemaining == "7" {
		urgencyColor = "#dc2626" // red for urgent
		urgencyText = "Urgent"
	} else if daysRemaining == "30" {
		urgencyColor = "#0066cc" // blue
		urgencyText = "Important"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AMC Expiry Notice</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); overflow: hidden; }
        .header { background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 30px; }
        .alert-badge { display: inline-block; background: %s; color: white; padding: 6px 16px; border-radius: 20px; font-size: 12px; font-weight: 600; margin-bottom: 20px; }
        .details { background: #e6f2ff; border: 1px solid #93c5fd; border-radius: 8px; padding: 20px; margin: 20px 0; }
        .detail-row { padding: 8px 0; border-bottom: 1px solid #bfdbfe; }
        .detail-row:last-child { border-bottom: none; }
        .expiry-box { background: linear-gradient(135deg, #e6f2ff 0%%, #bfdbfe 100%%); border-radius: 8px; padding: 20px; text-align: center; margin: 20px 0; }
        .expiry-days { font-size: 48px; font-weight: 700; color: %s; }
        .expiry-label { color: #1e40af; font-size: 14px; margin-top: 5px; }
        .button { display: inline-block; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: 600; background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%); color: white; margin-top: 20px; }
        .footer { background: #f9f9f9; padding: 20px; text-align: center; color: #666; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📋 AMC Expiry Notice</h1>
        </div>
        <div class="content">
            <span class="alert-badge">%s</span>
            <p>Dear %s,</p>
            <p>This is a reminder that your Annual Maintenance Contract (AMC) is expiring soon. Please contact our support team to renew and ensure uninterrupted service.</p>
            
            <div class="expiry-box">
                <div class="expiry-days">%s</div>
                <div class="expiry-label">Days Remaining</div>
            </div>
            
            <div class="details">
                <div class="detail-row"><strong>🛠️ Solution:</strong> %s</div>
                <div class="detail-row"><strong>📄 PO Number:</strong> %s</div>
                <div class="detail-row"><strong>📅 Expiry Date:</strong> %s</div>
            </div>
            
            <p style="color: #666; font-size: 13px; margin-top: 20px;">
                For renewal options, please contact our support team or reply to this email. We're here to help!
            </p>
            
            <a href="%s" class="button">View Details in Dashboard</a>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, urgencyColor, urgencyColor, urgencyText, customerName, daysRemaining, solutionName, poNumber, expiryDate, dashboardURL)
}

// WarrantyExpiryEmailTemplate - Template for Warranty expiry notifications
func WarrantyExpiryEmailTemplate(customerName, solutionName, poNumber, expiryDate, daysRemaining, dashboardURL string) string {
	urgencyColor := "#0066cc" // blue for info
	urgencyText := "Reminder"
	if daysRemaining == "7" {
		urgencyColor = "#dc2626" // red for urgent
		urgencyText = "Urgent"
	} else if daysRemaining == "30" {
		urgencyColor = "#0066cc" // blue
		urgencyText = "Important"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Warranty Expiry Notice</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); overflow: hidden; }
        .header { background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%); color: white; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; }
        .content { padding: 30px; }
        .alert-badge { display: inline-block; background: %s; color: white; padding: 6px 16px; border-radius: 20px; font-size: 12px; font-weight: 600; margin-bottom: 20px; }
        .details { background: #e6f2ff; border: 1px solid #93c5fd; border-radius: 8px; padding: 20px; margin: 20px 0; }
        .detail-row { padding: 8px 0; border-bottom: 1px solid #bfdbfe; }
        .detail-row:last-child { border-bottom: none; }
        .expiry-box { background: linear-gradient(135deg, #e6f2ff 0%%, #bfdbfe 100%%); border-radius: 8px; padding: 20px; text-align: center; margin: 20px 0; }
        .expiry-days { font-size: 48px; font-weight: 700; color: %s; }
        .expiry-label { color: #1e40af; font-size: 14px; margin-top: 5px; }
        .button { display: inline-block; background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%); color: white; padding: 12px 24px; text-decoration: none; border-radius: 8px; font-weight: 600; margin-top: 20px; }
        .footer { background: #f9f9f9; padding: 20px; text-align: center; color: #666; font-size: 12px; }
        .info-box { background: #e6f2ff; border-left: 4px solid #0066cc; padding: 15px; margin: 20px 0; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🛡️ Warranty Expiry Notice</h1>
        </div>
        <div class="content">
            <span class="alert-badge">%s</span>
            <p>Dear %s,</p>
            <p>This is a reminder that your product warranty is expiring soon. Please contact our support team to discuss renewal options.</p>
            
            <div class="expiry-box">
                <div class="expiry-days">%s</div>
                <div class="expiry-label">Days Remaining</div>
            </div>
            
            <div class="details">
                <div class="detail-row"><strong>🛠️ Solution:</strong> %s</div>
                <div class="detail-row"><strong>📄 PO Number:</strong> %s</div>
                <div class="detail-row"><strong>📅 Expiry Date:</strong> %s</div>
            </div>
            
            <div class="info-box">
                <strong>💡 Tip:</strong> Extend your warranty coverage or upgrade to an AMC plan for continued support.
            </div>
            
            <p style="color: #666; font-size: 13px; margin-top: 20px;">
                Contact our sales team for warranty renewal and AMC pricing options.
            </p>
            
            <a href="%s" class="button">View Details in Dashboard</a>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`, urgencyColor, urgencyColor, urgencyText, customerName, daysRemaining, solutionName, poNumber, expiryDate, dashboardURL)
}

// SLABreachEmailTemplate - Professional template for SLA breach notification
func SLABreachEmailTemplate(recipientName, ticketNumber, customerName, ticketTitle, priority string, hoursOverdue, slaHours int, dashboardURL string) string {
	// Color based on priority
	urgencyColor := "#0066cc" // blue for Standard
	if priority == "Critical" {
		urgencyColor = "#dc2626" // red for Critical
	} else if priority == "Standard" {
		urgencyColor = "#0066cc" // blue for Standard
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SLA Breach Notification - Immediate Action Required</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, %s 0%%, %s 100%%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 700;
        }
        .header p {
            margin: 10px 0 0 0;
            font-size: 14px;
            opacity: 0.9;
        }
        .alert-banner {
            background-color: %s;
            color: white;
            padding: 20px;
            text-align: center;
            font-weight: 600;
            font-size: 16px;
        }
        .content {
            padding: 40px 30px;
        }
        .greeting {
            font-size: 16px;
            color: #333;
            margin-bottom: 20px;
            font-weight: 500;
        }
        .message {
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .ticket-details {
            background-color: #f9f9f9;
            border-left: 4px solid %s;
            padding: 20px;
            margin: 25px 0;
            border-radius: 4px;
        }
        .detail-row {
            display: flex;
            justify-content: space-between;
            margin-bottom: 12px;
            font-size: 14px;
        }
        .detail-label {
            font-weight: 600;
            color: #333;
            min-width: 120px;
        }
        .detail-value {
            color: #666;
            text-align: right;
            flex: 1;
        }
        .breach-info {
            background-color: #e6f2ff;
            border-left: 4px solid #0066cc;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 14px;
            color: #0052a3;
        }
        .action-required {
            background-color: #fee2e2;
            border-left: 4px solid #dc2626;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 14px;
            color: #7f1d1d;
            font-weight: 500;
        }
        .button-container {
            text-align: center;
            margin: 30px 0;
        }
        .button {
            display: inline-block;
            background: linear-gradient(135deg, %s 0%%, %s 100%%);
            color: white;
            padding: 14px 32px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 15px;
        }
        .footer {
            background-color: #f9f9f9;
            padding: 20px;
            text-align: center;
            color: #999;
            font-size: 12px;
            border-top: 1px solid #eee;
        }
        .timeline {
            background-color: #f9f9f9;
            border-left: 4px solid %s;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 13px;
        }
        .timeline-item {
            margin-bottom: 8px;
            color: #666;
        }
        .timeline-item strong {
            color: #333;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚠️ SLA Breach Alert</h1>
            <p>Immediate action required</p>
        </div>
        <div class="alert-banner">
            Ticket SLA deadline has been exceeded - %d hours overdue
        </div>
        <div class="content">
            <div class="greeting">Hello %s,</div>
            <div class="message">
                This is a critical notification. The support ticket listed below has exceeded its Service Level Agreement (SLA) deadline. Immediate attention and action are required.
            </div>

            <div class="ticket-details">
                <div class="detail-row">
                    <span class="detail-label">Ticket #:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Title:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Customer:</span>
                    <span class="detail-value">%s</span>
                </div>
                <div class="detail-row">
                    <span class="detail-label">Priority:</span>
                    <span class="detail-value"><span style="color: %s; font-weight: 600;">%s</span></span>
                </div>
            </div>

            <div class="action-required">
                <strong>⚠️ ACTION REQUIRED:</strong> This ticket has breached its SLA by %d hours. Please take immediate action to resolve or escalate this ticket.
            </div>

            <div class="timeline">
                <div class="timeline-item"><strong>SLA Duration:</strong> %d hours (%d days)</div>
                <div class="timeline-item"><strong>Overdue By:</strong> %d hours</div>
                <div class="timeline-item"><strong>Status:</strong> <span style="color: #dc2626; font-weight: 600;">BREACHED</span></div>
            </div>

            <div class="button-container">
                <a href="%s" class="button">View Ticket Details</a>
            </div>

            <div class="message" style="margin-top: 30px; padding-top: 30px; border-top: 1px solid #eee;">
                <strong>What this means:</strong><br>
                Your support team has failed to meet the agreed-upon response time for this ticket. This may impact customer satisfaction and your SLA metrics. Please prioritize this ticket accordingly.
            </div>

            <div class="message" style="font-size: 13px; color: #999;">
                <strong>Next Steps:</strong><br>
                • Immediately review the ticket details<br>
                • Contact the assigned support engineer if needed<br>
                • Update the ticket status and add a comment explaining the delay<br>
                • If resolution is not immediate, escalate to management<br>
                • Document the reason for SLA breach in ticket comments
            </div>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
            <p>This is an automated notification. Please do not reply to this email.</p>
        </div>
    </div>
</body>
</html>`, urgencyColor, urgencyColor, urgencyColor, urgencyColor, urgencyColor, urgencyColor, urgencyColor, hoursOverdue, recipientName, ticketNumber, ticketTitle, customerName, urgencyColor, priority, hoursOverdue, slaHours, slaHours/24, hoursOverdue, dashboardURL)
}

// TicketClosureEmailTemplate - Professional template for ticket closure notification
func TicketClosureEmailTemplate(customerName, ticketNumber, ticketTitle, engineerName, closureDate, closureComment, dashboardURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ticket Closed - Support Ticket #%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            margin: 0;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 40px 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 28px;
            font-weight: 700;
        }
        .header p {
            margin: 10px 0 0 0;
            font-size: 16px;
            opacity: 0.9;
        }
        .content {
            padding: 40px 30px;
        }
        .ticket-info {
            background-color: #e6f2ff;
            border: 2px solid #0066cc;
            border-radius: 6px;
            padding: 20px;
            margin-bottom: 30px;
        }
        .ticket-info-row {
            display: flex;
            justify-content: space-between;
            padding: 10px 0;
            border-bottom: 1px solid #bfdbfe;
        }
        .ticket-info-row:last-child {
            border-bottom: none;
        }
        .ticket-info-label {
            font-weight: 600;
            color: #0052a3;
            font-size: 14px;
        }
        .ticket-info-value {
            color: #0052a3;
            font-size: 14px;
            text-align: right;
            word-break: break-word;
        }
        .status-badge {
            display: inline-block;
            background-color: #0066cc;
            color: white;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 13px;
            font-weight: 600;
        }
        .greeting {
            font-size: 16px;
            color: #333;
            margin-bottom: 20px;
        }
        .message {
            color: #666;
            margin-bottom: 20px;
            font-size: 14px;
            line-height: 1.8;
        }
        .closure-comment {
            background-color: #f9fafb;
            border-left: 4px solid #0066cc;
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
            font-size: 14px;
            color: #374151;
        }
        .closure-comment-label {
            font-weight: 600;
            color: #0066cc;
            font-size: 13px;
            text-transform: uppercase;
            margin-bottom: 8px;
        }
        .button-container {
            text-align: center;
            margin: 30px 0;
        }
        .button {
            display: inline-block;
            background: linear-gradient(135deg, #0066cc 0%%, #0052a3 100%%);
            color: white;
            padding: 14px 32px;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 600;
            font-size: 15px;
        }
        .footer {
            background-color: #f9f9f9;
            padding: 20px;
            text-align: center;
            color: #999;
            font-size: 12px;
            border-top: 1px solid #eee;
        }
        .footer p {
            margin: 5px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>✓ Your Support Ticket Has Been Resolved</h1>
            <p>Ticket #%s</p>
        </div>
        <div class="content">
            <div class="greeting">Hello %s,</div>
            
            <div class="message">
                We're pleased to inform you that your support ticket has been successfully closed. Our team has completed the resolution and your issue should now be fully resolved.
            </div>

            <div class="ticket-info">
                <div class="ticket-info-row">
                    <span class="ticket-info-label">Ticket Number:</span>
                    <span class="ticket-info-value">#%s</span>
                </div>
                <div class="ticket-info-row">
                    <span class="ticket-info-label">Ticket Title:</span>
                    <span class="ticket-info-value">%s</span>
                </div>
                <div class="ticket-info-row">
                    <span class="ticket-info-label">Resolved by:</span>
                    <span class="ticket-info-value">%s</span>
                </div>
                <div class="ticket-info-row">
                    <span class="ticket-info-label">Closure Date:</span>
                    <span class="ticket-info-value">%s</span>
                </div>
                <div class="ticket-info-row">
                    <span class="ticket-info-label">Status:</span>
                    <span class="ticket-info-value"><span class="status-badge">✓ CLOSED</span></span>
                </div>
            </div>

            <div class="closure-comment">
                <div class="closure-comment-label">Support Comment:</div>
                <div>%s</div>
            </div>

            <div class="message">
                <strong>What Next?</strong><br>
                If you experience any further issues related to this ticket or have any questions about the resolution, please don't hesitate to contact our support team. We're here to help!
            </div>

            <div class="button-container">
                <a href="%s" class="button">View Your Ticket Details</a>
            </div>

            <div class="message" style="font-size: 13px; color: #999; margin-top: 30px; padding-top: 30px; border-top: 1px solid #eee;">
                <strong>Thank you for choosing our service!</strong><br>
                We value your feedback. If you'd like to leave feedback about your support experience, please visit your dashboard.
            </div>
        </div>
        <div class="footer">
            <p>© 2026 Vsmart. All rights reserved.</p>
            <p>This is an automated notification. Please do not reply to this email.</p>
            <p>For immediate assistance, contact our support team through your dashboard.</p>
        </div>
    </div>
</body>
</html>`, ticketNumber, ticketNumber, customerName, ticketNumber, ticketTitle, engineerName, closureDate, closureComment, dashboardURL)
}
