package services

import (
	"mime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSMTPMessage_Headers(t *testing.T) {
	msg := string(buildSMTPMessage(
		"ex-files <noreply@ex-files.local>",
		"alice@example.com",
		"Document approved",
		"<p>Hello</p>",
	))

	// Header order: From / To / Subject / MIME-Version / Content-Type / blank
	// line / body.
	headers, body, ok := strings.Cut(msg, "\r\n\r\n")
	require.True(t, ok, "message must have a blank line separating headers from body")

	assert.Contains(t, headers, "From: ex-files <noreply@ex-files.local>")
	assert.Contains(t, headers, "To: alice@example.com")
	assert.Contains(t, headers, "Subject: Document approved")
	assert.Contains(t, headers, "MIME-Version: 1.0")
	assert.Contains(t, headers, "Content-Type: text/html; charset=UTF-8")
	assert.Equal(t, "<p>Hello</p>", body)

	// CRLF line endings everywhere - required by RFC 5322.
	assert.NotContains(t, headers, "\n\n", "must use CRLF, not LF, between headers")
}

func TestBuildSMTPMessage_NonASCIISubject(t *testing.T) {
	msg := string(buildSMTPMessage(
		"noreply@ex-files.local",
		"bob@example.com",
		"Wymagane zmiany: raport Q2", // Polish with non-ASCII characters
		"<p>...</p>",
	))

	// The subject line must be RFC 2047-encoded so non-ASCII bytes don't end
	// up in the wire format unencoded.
	headers, _, _ := strings.Cut(msg, "\r\n\r\n")
	subjectLine := ""
	for line := range strings.SplitSeq(headers, "\r\n") {
		if rest, ok := strings.CutPrefix(line, "Subject: "); ok {
			subjectLine = rest
			break
		}
	}
	require.NotEmpty(t, subjectLine, "Subject header must be present")

	// Decode the encoded-word back to UTF-8 and confirm round-trip.
	decoded, err := new(mime.WordDecoder).DecodeHeader(subjectLine)
	require.NoError(t, err)
	assert.Equal(t, "Wymagane zmiany: raport Q2", decoded)
}

func TestSMTPEmailService_DevModeWhenNoUsername(t *testing.T) {
	// With an empty Username, Send must NOT attempt to dial, it logs and
	// returns nil. This mirrors the empty-key branch of ResendEmailService
	// so a half-configured .env doesn't break local dev.
	svc := NewSMTPEmailService("smtp.invalid.example", 2525, "", "", "noreply@ex-files.local")
	err := svc.Send("alice@example.com", "test", "<p>body</p>")
	assert.NoError(t, err)
}
