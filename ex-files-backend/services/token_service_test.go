package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

func TestJWTTokenService_IssueAndValidate(t *testing.T) {
	tests := []struct {
		name string
		user *models.User
	}{
		{
			name: "manager user",
			user: &models.User{
				Model: gorm.Model{ID: 1},
				Email: "alice@example.com",
				Role:  models.RoleManager,
			},
		},
		{
			name: "employee user",
			user: &models.User{
				Model: gorm.Model{ID: 42},
				Email: "bob@example.com",
				Role:  models.RoleEmployee,
			},
		},
		{
			name: "root user",
			user: &models.User{
				Model: gorm.Model{ID: 99},
				Email: "root@example.com",
				Role:  models.RoleRoot,
			},
		},
	}

	svc := services.NewJWTTokenService("test-secret")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.Issue(tt.user)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			claims, err := svc.Validate(token)
			require.NoError(t, err)
			assert.Equal(t, tt.user.ID, claims.UserID)
			assert.Equal(t, tt.user.Email, claims.Email)
			assert.Equal(t, tt.user.Role, claims.Role)
		})
	}
}

func TestJWTTokenService_ValidateWrongKey(t *testing.T) {
	issuer := services.NewJWTTokenService("secret1")
	validator := services.NewJWTTokenService("secret2")

	user := &models.User{
		Model: gorm.Model{ID: 1},
		Email: "alice@example.com",
		Role:  models.RoleEmployee,
	}

	token, err := issuer.Issue(user)
	require.NoError(t, err)

	_, err = validator.Validate(token)
	assert.Error(t, err)
}

func TestJWTTokenService_ValidateInvalidTokens(t *testing.T) {
	svc := services.NewJWTTokenService("test-secret")

	tests := []struct {
		name     string
		tokenStr string
	}{
		{name: "malformed token", tokenStr: "not.a.valid.jwt.token"},
		{name: "empty token", tokenStr: ""},
		{name: "random garbage", tokenStr: "abc123!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := svc.Validate(tt.tokenStr)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestJWTTokenService_IssueDifferentTokensForDifferentUsers(t *testing.T) {
	svc := services.NewJWTTokenService("test-secret")

	user1 := &models.User{
		Model: gorm.Model{ID: 1},
		Email: "alice@example.com",
		Role:  models.RoleManager,
	}
	user2 := &models.User{
		Model: gorm.Model{ID: 2},
		Email: "bob@example.com",
		Role:  models.RoleEmployee,
	}

	token1, err := svc.Issue(user1)
	require.NoError(t, err)

	token2, err := svc.Issue(user2)
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)
}
