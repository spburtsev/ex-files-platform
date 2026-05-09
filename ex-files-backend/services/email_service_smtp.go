package services

import (
	"bytes"
	"fmt"
	"log/slog"
	"mime"
	"net/smtp"
	"strconv"
)

// SMTPEmailService sends mail through any SMTP server. The dev sandbox at
// sandbox.smtp.mailtrap.io:2525 is the primary target, but any SMTP backend
// (Mailpit, SES, an internal relay) works as long as it accepts AUTH PLAIN
// over a STARTTLS-upgraded connection.
//
// When Username is empty the service skips dialing and logs the message
// instead, same dev fallback as ResendEmailService, so a missing .env
// doesn't crash the boot path.
type SMTPEmailService struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func NewSMTPEmailService(host string, port int, username, password, from string) *SMTPEmailService {
	return &SMTPEmailService{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

// buildSMTPMessage formats an RFC 5322 message body suitable for handing to
// smtp.SendMail. Subject is RFC 2047-encoded so non-ASCII (e.g. Polish
// translations from messages/pl.json) round-trips correctly.
func buildSMTPMessage(from, to, subject, htmlBody string) []byte {
	var b bytes.Buffer
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\n")
	b.WriteString("To: ")
	b.WriteString(to)
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(mime.QEncoding.Encode("UTF-8", subject))
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return b.Bytes()
}

func (s *SMTPEmailService) Send(to, subject, body string) error {
	if s.Username == "" {
		slog.Info("email (dev mode, not sent)",
			"to", to, "subject", subject, "body_length", len(body))
		return nil
	}

	addr := s.Host + ":" + strconv.Itoa(s.Port)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	msg := buildSMTPMessage(s.From, to, subject, body)

	if err := smtp.SendMail(addr, auth, s.From, []string{to}, msg); err != nil {
		slog.Error("smtp send failed",
			"host", s.Host, "port", s.Port, "to", to, "subject", subject, "error", err)
		return fmt.Errorf("smtp send: %w", err)
	}

	slog.Info("email sent", "to", to, "subject", subject)
	return nil
}
