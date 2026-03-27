package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanManageWorkspaces(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{
			name: "root_can_manage",
			role: RoleRoot,
			want: true,
		},
		{
			name: "manager_can_manage",
			role: RoleManager,
			want: true,
		},
		{
			name: "employee_cannot_manage",
			role: RoleEmployee,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.role.CanManageWorkspaces()
			assert.Equal(t, tc.want, got)
		})
	}
}
