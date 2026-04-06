package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/spburtsev/ex-files-backend/services"
)

func TestBcryptHasher_HashAndCompare(t *testing.T) {
	hasher := services.BcryptHasher{Cost: bcrypt.MinCost}

	tests := []struct {
		name     string
		password string
	}{
		{name: "simple password", password: "password123"},
		{name: "complex password", password: "P@$$w0rd!#&*()"},
		{name: "unicode password", password: "пароль日本語"},
		{name: "long password", password: "aVeryLongPasswordThatExceedsTypicalLengthRequirements1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.password, hash)

			err = hasher.Compare(hash, tt.password)
			assert.NoError(t, err)
		})
	}
}

func TestBcryptHasher_CompareWrongPassword(t *testing.T) {
	hasher := services.BcryptHasher{Cost: bcrypt.MinCost}

	tests := []struct {
		name          string
		realPassword  string
		wrongPassword string
	}{
		{name: "completely different", realPassword: "correct", wrongPassword: "wrong"},
		{name: "off by one char", realPassword: "password1", wrongPassword: "password2"},
		{name: "extra character", realPassword: "secret", wrongPassword: "secrets"},
		{name: "empty vs non-empty", realPassword: "notempty", wrongPassword: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.realPassword)
			require.NoError(t, err)

			err = hasher.Compare(hash, tt.wrongPassword)
			assert.Error(t, err)
		})
	}
}

func TestBcryptHasher_HashProducesDifferentOutputs(t *testing.T) {
	hasher := services.BcryptHasher{Cost: bcrypt.MinCost}
	password := "same-password"

	hash1, err := hasher.Hash(password)
	require.NoError(t, err)

	hash2, err := hasher.Hash(password)
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "bcrypt should produce different hashes due to random salt")
}

func TestBcryptHasher_CompareWithEmptyHash(t *testing.T) {
	hasher := services.BcryptHasher{Cost: bcrypt.MinCost}

	err := hasher.Compare("", "anypassword")
	assert.Error(t, err)
}
