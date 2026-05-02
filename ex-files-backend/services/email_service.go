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
// When APIKey is empty, it logs emails instead of sending them (dev mode).
type ResendEmailService struct {
	APIKey  string
	From    string
	BaseURL string
}

func NewResendEmailService(apiKey, from string) *ResendEmailService {
	baseURL := "https://api.resend.com"
	return &ResendEmailService{APIKey: apiKey, From: from, BaseURL: baseURL}
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (r *ResendEmailService) Send(to, subject, body string) error {
	if r.APIKey == "" {
		slog.Info("email (dev mode, not sent)",
			"to", to, "subject", subject, "body_length", len(body))
		return nil
	}

	payload := resendPayload{
		From:    r.From,
		To:      []string{to},
		Subject: subject,
		HTML:    body,
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
