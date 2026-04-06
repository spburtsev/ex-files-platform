package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOwnedBy(t *testing.T) {
	tests := []struct {
		name      string
		managerID uint
		userID    uint
		want      bool
	}{
		{
			name:      "matching_manager_id",
			managerID: 42,
			userID:    42,
			want:      true,
		},
		{
			name:      "non_matching_manager_id",
			managerID: 42,
			userID:    99,
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ws := &Workspace{ManagerID: tc.managerID}
			got := ws.IsOwnedBy(tc.userID)
			assert.Equal(t, tc.want, got)
		})
	}
}
