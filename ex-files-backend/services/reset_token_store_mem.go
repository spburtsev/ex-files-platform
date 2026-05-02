package services

import (
	"fmt"
	"sync"
	"time"
)

// InMemoryResetTokenStore is a fallback ResetTokenStore for development without Redis.
type InMemoryResetTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]resetEntry
}

type resetEntry struct {
	userID    uint
	expiresAt time.Time
}

func NewInMemoryResetTokenStore() *InMemoryResetTokenStore {
	return &InMemoryResetTokenStore{tokens: make(map[string]resetEntry)}
}

func (s *InMemoryResetTokenStore) StoreResetToken(token string, userID uint, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = resetEntry{userID: userID, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (s *InMemoryResetTokenStore) GetResetTokenUserID(token string) (uint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.tokens[token]
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, fmt.Errorf("token not found or expired")
	}
	return entry.userID, nil
}

func (s *InMemoryResetTokenStore) DeleteResetToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
	return nil
}
