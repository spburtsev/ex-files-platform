package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		current DocumentStatus
		next    DocumentStatus
		want    bool
	}{
		// From pending
		{
			name:    "pending_to_in_review",
			current: DocumentStatusPending,
			next:    DocumentStatusInReview,
			want:    true,
		},
		{
			name:    "pending_to_approved",
			current: DocumentStatusPending,
			next:    DocumentStatusApproved,
			want:    false,
		},
		{
			name:    "pending_to_rejected",
			current: DocumentStatusPending,
			next:    DocumentStatusRejected,
			want:    false,
		},

		// From in_review
		{
			name:    "in_review_to_approved",
			current: DocumentStatusInReview,
			next:    DocumentStatusApproved,
			want:    true,
		},
		{
			name:    "in_review_to_rejected",
			current: DocumentStatusInReview,
			next:    DocumentStatusRejected,
			want:    true,
		},
		{
			name:    "in_review_to_changes_requested",
			current: DocumentStatusInReview,
			next:    DocumentStatusChangesRequested,
			want:    true,
		},
		{
			name:    "in_review_to_pending",
			current: DocumentStatusInReview,
			next:    DocumentStatusPending,
			want:    false,
		},

		// From changes_requested
		{
			name:    "changes_requested_to_in_review",
			current: DocumentStatusChangesRequested,
			next:    DocumentStatusInReview,
			want:    true,
		},
		{
			name:    "changes_requested_to_approved",
			current: DocumentStatusChangesRequested,
			next:    DocumentStatusApproved,
			want:    false,
		},

		// Terminal states
		{
			name:    "approved_to_pending",
			current: DocumentStatusApproved,
			next:    DocumentStatusPending,
			want:    false,
		},
		{
			name:    "approved_to_in_review",
			current: DocumentStatusApproved,
			next:    DocumentStatusInReview,
			want:    false,
		},
		{
			name:    "rejected_to_pending",
			current: DocumentStatusRejected,
			next:    DocumentStatusPending,
			want:    false,
		},
		{
			name:    "rejected_to_in_review",
			current: DocumentStatusRejected,
			next:    DocumentStatusInReview,
			want:    false,
		},

		// Unknown status
		{
			name:    "unknown_to_pending",
			current: DocumentStatus("unknown"),
			next:    DocumentStatusPending,
			want:    false,
		},
		{
			name:    "unknown_to_in_review",
			current: DocumentStatus("unknown"),
			next:    DocumentStatusInReview,
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := &Document{Status: tc.current}
			got := doc.CanTransitionTo(tc.next)
			assert.Equal(t, tc.want, got)
		})
	}
}
