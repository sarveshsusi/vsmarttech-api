package utils

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"rbac/config"

	"github.com/google/uuid"
)

type Mailer struct {
	auth smtp.Auth
	host string
	port int
	from string
}

func NewMailer(cfg config.MailConfig) *Mailer {
	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		return nil
	}
	auth := smtp.PlainAuth(
		"",
		cfg.Username,
		cfg.Password,
		cfg.Host,
	)

	return &Mailer{
		auth: auth,
		host: cfg.Host,
		port: cfg.Port,
		from: cfg.From,
	}
}

func stripHTML(html string) string {
	plain := html
	replacements := []string{
		"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"</p>", "\n\n", "</div>", "\n", "</h1>", "\n", "</li>", "\n",
	}
	for i := 0; i+1 < len(replacements); i += 2 {
		plain = strings.ReplaceAll(plain, replacements[i], replacements[i+1])
	}
	var b strings.Builder
	inTag := false
	for _, r := range plain {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func (m *Mailer) Send(to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	boundary := "vsmart-boundary-" + uuid.New().String()
	plain := stripHTML(htmlBody)
	if plain == "" {
		plain = "Please view this message in an HTML-capable email client."
	}

	msg := []byte(
		"From: " + m.from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
			"Message-ID: <" + uuid.New().String() + "@vsmart.app>\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n" +
			"X-Mailer: Vsmart CRM\r\n" +
			"X-Priority: 3\r\n" +
			"Return-Path: " + m.from + "\r\n\r\n" +

			"--" + boundary + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n\r\n" +
			plain + "\r\n\r\n" +

			"--" + boundary + "\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n\r\n" +
			htmlBody + "\r\n\r\n" +

			"--" + boundary + "--",
	)

	return smtp.SendMail(addr, m.auth, m.from, []string{to}, msg)
}
