package utils

import (
	"fmt"
	"net/smtp"
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

func (m *Mailer) Send(to, subject, html string) error {
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	boundary := "rbac-boundary"

	msg := []byte(
		"From: " + m.from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
			"Message-ID: <" + uuid.New().String() + "@vsmart.app>\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: multipart/alternative; boundary=" + boundary + "\r\n" +
			"X-Mailer: Vsmart\r\n" +
			"X-Priority: 3\r\n" +
			"Return-Path: " + m.from + "\r\n\r\n" +

			"--" + boundary + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n\r\n" +
			"Please open this email in HTML format to reset your password.\r\n\r\n" +

			"--" + boundary + "\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n\r\n" +
			"<html><body>" + html + "</body></html>\r\n\r\n" +

			"--" + boundary + "--",
	)

	return smtp.SendMail(addr, m.auth, m.from, []string{to}, msg)
}
