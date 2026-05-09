package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// ResendEmailService sends emails via the Resend API (https://resend.com).
type ResendEmailService struct {
	APIKey  string
	From    string
	BaseURL string
	DevTrap string
}

func NewResendEmailService(apiKey, from, devTrap string) *ResendEmailService {
	baseURL := "https://api.resend.com"
	return &ResendEmailService{APIKey: apiKey, From: from, BaseURL: baseURL, DevTrap: devTrap}
}

type resendPayload struct {
	From    string            `json:"from"`
	To      []string          `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (r *ResendEmailService) Send(to, subject, body string) error {
	if r.APIKey == "" {
		slog.Info("email (dev mode, not sent)",
			"to", to, "subject", subject, "body_length", len(body))
		return nil
	}

	originalTo := to
	headers := map[string]string{}
	if r.DevTrap != "" && r.DevTrap != to {
		slog.Info("email rerouted to RESEND_DEV_TRAP",
			"original_to", originalTo, "trap", r.DevTrap, "subject", subject)
		to = r.DevTrap
		subject = "[trap → " + originalTo + "] " + subject
		headers["X-Original-To"] = originalTo
	}

	payload := resendPayload{
		From:    r.From,
		To:      []string{to},
		Subject: subject,
		HTML:    body,
	}
	if len(headers) > 0 {
		payload.Headers = headers
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.BaseURL+"/emails", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create email request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send email request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("resend API error",
			"status", resp.StatusCode, "body", string(respBody),
			"to", to, "subject", subject)
		return fmt.Errorf("resend API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	slog.Info("email sent", "to", to, "subject", subject)
	return nil
}
