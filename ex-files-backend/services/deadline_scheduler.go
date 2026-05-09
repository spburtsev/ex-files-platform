package services

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// DeadlineScheduler periodically scans unresolved issues with deadlines and
// broadcasts SSE reminders targeted at each issue's assignee as the deadline
// crosses thresholds (24h, 1h, overdue).
//
// Crossings are tracked in-memory; restarting the process resets state, so a
// running issue may receive at most one duplicate notification per surviving
// threshold across a deploy. That's the trade-off chosen for simplicity over
// adding persistence (Redis SETNX would be the upgrade path).
type DeadlineScheduler struct {
	Hub       *SSEHub
	IssueRepo IssueRepository
	Tick      time.Duration

	mu       sync.Mutex
	notified map[uint]map[string]bool // issueID → threshold → fired?
}

const (
	thresholdOverdue = "overdue"
	threshold1h      = "1h"
	threshold24h     = "24h"
)

// Run blocks until ctx is cancelled. Intended to be invoked in its own
// goroutine from main.
func (s *DeadlineScheduler) Run(ctx context.Context) {
	if s.Tick <= 0 {
		s.Tick = 10 * time.Minute
	}
	s.mu.Lock()
	if s.notified == nil {
		s.notified = make(map[uint]map[string]bool)
	}
	s.mu.Unlock()

	t := time.NewTicker(s.Tick)
	defer t.Stop()

	// Run once immediately so a deploy doesn't wait a full Tick before the
	// first scan.
	s.scan()

	for {
		select {
		case <-ctx.Done():
			slog.Info("deadline scheduler stopping")
			return
		case <-t.C:
			s.scan()
		}
	}
}

func (s *DeadlineScheduler) scan() {
	if s.Hub == nil || s.IssueRepo == nil {
		return
	}
	issues, err := s.IssueRepo.ListUnresolvedWithDeadline()
	if err != nil {
		slog.Error("deadline scheduler list failed", "component", "deadline", "error", err)
		return
	}
	now := time.Now()
	for _, iss := range issues {
		if iss.Deadline == nil {
			continue
		}
		threshold := classify(now, *iss.Deadline)
		if threshold == "" {
			continue
		}
		if !s.markFired(iss.ID, threshold) {
			continue // already notified for this threshold
		}
		s.Hub.Broadcast(SSEEvent{
			Type:   "deadline.approaching",
			UserID: iss.AssigneeID,
			Payload: map[string]any{
				"issue_id":     iss.ID,
				"workspace_id": iss.WorkspaceID,
				"title":        iss.Title,
				"deadline":     iss.Deadline,
				"threshold":    threshold,
			},
		})
	}
}

// classify returns the highest-urgency threshold that the deadline currently
// occupies, or "" if it's still beyond the 24h horizon. We always fire the
// most urgent crossing so callers can present a single toast per issue per
// scan (priority: overdue > 1h > 24h).
func classify(now, deadline time.Time) string {
	if deadline.Before(now) {
		return thresholdOverdue
	}
	d := deadline.Sub(now)
	if d <= time.Hour {
		return threshold1h
	}
	if d <= 24*time.Hour {
		return threshold24h
	}
	return ""
}

// markFired returns true the first time we see (issueID, threshold): that's
// the only call that broadcasts. Subsequent calls in the same process return
// false until the in-memory map is reset.
func (s *DeadlineScheduler) markFired(issueID uint, threshold string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	bucket, ok := s.notified[issueID]
	if !ok {
		bucket = make(map[string]bool)
		s.notified[issueID] = bucket
	}
	if bucket[threshold] {
		return false
	}
	bucket[threshold] = true
	return true
}
