package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryResetTokenStore_StoreAndGet(t *testing.T) {
	store := NewInMemoryResetTokenStore()

	require.NoError(t, store.StoreResetToken("tok123", 42, 1*time.Hour))

	userID, err := store.GetResetTokenUserID("tok123")
	require.NoError(t, err)
	assert.Equal(t, uint(42), userID)
}

func TestInMemoryResetTokenStore_NotFound(t *testing.T) {
	store := NewInMemoryResetTokenStore()

	_, err := store.GetResetTokenUserID("nonexistent")
	assert.Error(t, err)
}

func TestInMemoryResetTokenStore_Expired(t *testing.T) {
	store := NewInMemoryResetTokenStore()

	require.NoError(t, store.StoreResetToken("tok", 1, 1*time.Millisecond))
	time.Sleep(5 * time.Millisecond)

	_, err := store.GetResetTokenUserID("tok")
	assert.Error(t, err)
}

func TestInMemoryResetTokenStore_Delete(t *testing.T) {
	store := NewInMemoryResetTokenStore()

	store.StoreResetToken("tok", 1, 1*time.Hour)
	require.NoError(t, store.DeleteResetToken("tok"))

	_, err := store.GetResetTokenUserID("tok")
	assert.Error(t, err)
}

func TestInMemoryResetTokenStore_Overwrite(t *testing.T) {
	store := NewInMemoryResetTokenStore()

	store.StoreResetToken("tok", 1, 1*time.Hour)
	store.StoreResetToken("tok", 2, 1*time.Hour)

	userID, err := store.GetResetTokenUserID("tok")
	require.NoError(t, err)
	assert.Equal(t, uint(2), userID)
}
