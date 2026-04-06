package services

import (
	"fmt"
	"log/slog"
	"net/smtp"
)

type SMTPEmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewSMTPEmailService(host string, port int, user, password, from string) *SMTPEmailService {
	return &SMTPEmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

func (s *SMTPEmailService) Send(to, subject, body string) error {
	if s.host == "" {
		slog.Warn("SMTP not configured, skipping send", "component", "email", "to", to)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.from, to, subject, body)

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
