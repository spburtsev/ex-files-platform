package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResendEmail_DevMode(t *testing.T) {
	svc := NewResendEmailService("", "test@example.com")
	err := svc.Send("user@example.com", "Test Subject", "<p>Hello</p>")
	assert.NoError(t, err, "dev mode should not return an error")
}

func TestResendEmail_Success(t *testing.T) {
	var receivedPayload resendPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/emails", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"email-123"}`))
	}))
	defer server.Close()

	svc := &ResendEmailService{
		APIKey:  "test-api-key",
		From:    "noreply@test.com",
		BaseURL: server.URL,
	}

	err := svc.Send("recipient@test.com", "Test Subject", "<p>Hello</p>")
	assert.NoError(t, err)
	assert.Equal(t, "noreply@test.com", receivedPayload.From)
	assert.Equal(t, []string{"recipient@test.com"}, receivedPayload.To)
	assert.Equal(t, "Test Subject", receivedPayload.Subject)
	assert.Equal(t, "<p>Hello</p>", receivedPayload.HTML)
}

func TestResendEmail_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Invalid API key"}`))
	}))
	defer server.Close()

	svc := &ResendEmailService{
		APIKey:  "bad-key",
		From:    "noreply@test.com",
		BaseURL: server.URL,
	}

	err := svc.Send("recipient@test.com", "Subject", "Body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestResendEmail_ServerDown(t *testing.T) {
	svc := &ResendEmailService{
		APIKey:  "test-key",
		From:    "noreply@test.com",
		BaseURL: "http://localhost:1", // unlikely to be listening
	}

	err := svc.Send("recipient@test.com", "Subject", "Body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send email request")
}
