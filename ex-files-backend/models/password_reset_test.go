package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPasswordResetToken_IsExpired(t *testing.T) {
	t.Run("not_expired", func(t *testing.T) {
		token := PasswordResetToken{
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		assert.False(t, token.IsExpired())
	})

	t.Run("expired", func(t *testing.T) {
		token := PasswordResetToken{
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		assert.True(t, token.IsExpired())
	})

	t.Run("just_expired", func(t *testing.T) {
		token := PasswordResetToken{
			ExpiresAt: time.Now().Add(-1 * time.Millisecond),
		}
		assert.True(t, token.IsExpired())
	})
}
