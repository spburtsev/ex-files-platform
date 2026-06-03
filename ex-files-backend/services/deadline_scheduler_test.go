package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeadlineScheduler_classify(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		deadline time.Time
		want     string
	}{
		{"overdue", now.Add(-time.Minute), thresholdOverdue},
		{"within 1h", now.Add(30 * time.Minute), threshold1h},
		{"exactly 1h", now.Add(time.Hour), threshold1h},
		{"within 24h", now.Add(10 * time.Hour), threshold24h},
		{"exactly 24h", now.Add(24 * time.Hour), threshold24h},
		{"beyond 24h", now.Add(48 * time.Hour), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, classify(now, c.deadline))
		})
	}
}

func TestDeadlineScheduler_markFired(t *testing.T) {
	s := &DeadlineScheduler{notified: make(map[uint]map[string]bool)}

	assert.True(t, s.markFired(1, threshold24h), "first crossing for (1, 24h) fires")
	assert.False(t, s.markFired(1, threshold24h), "duplicate is suppressed")
	assert.True(t, s.markFired(1, threshold1h), "a different threshold on the same issue fires")
	assert.True(t, s.markFired(2, threshold24h), "a different issue fires independently")
}
