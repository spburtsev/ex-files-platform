package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func verifyServer(repo *mockDocumentRepo) *handlers.Server {
	return &handlers.Server{
		UserRepo:     &mockUserRepo{},
		Tokens:       &mockTokens{},
		Hasher:       stubHasher{},
		Audit:        &dummyAudit{},
		DocumentRepo: repo,
	}
}

func TestVerify_Found(t *testing.T) {
	repo := &mockDocumentRepo{}
	createdAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	repo.On("FindByHash", "abc123").Return(&models.Document{
		Model:  gorm.Model{ID: 1, CreatedAt: createdAt, UpdatedAt: createdAt},
		Name:   "report.pdf",
		Hash:   "abc123",
		Status: models.DocumentStatusApproved,
	}, nil)

	srv := newTestServer(t, verifyServer(repo))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/verify?hash=abc123")
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.VerifyResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.True(t, got.Verified)
	require.True(t, got.DocumentName.IsSet())
	assert.Equal(t, "report.pdf", got.DocumentName.Value)
	require.True(t, got.Status.IsSet())
	assert.Equal(t, oapi.DocumentStatusApproved, got.Status.Value)
	require.True(t, got.Hash.IsSet())
	assert.Equal(t, "abc123", got.Hash.Value)
}

func TestVerify_NotFound(t *testing.T) {
	repo := &mockDocumentRepo{}
	repo.On("FindByHash", "deadbeef").Return(nil, gorm.ErrRecordNotFound)

	srv := newTestServer(t, verifyServer(repo))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/verify?hash=deadbeef")
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.VerifyResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.False(t, got.Verified)
	assert.False(t, got.DocumentName.IsSet())
}

func TestVerify_MissingHashReturns400(t *testing.T) {
	srv := newTestServer(t, verifyServer(&mockDocumentRepo{}))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/verify")
	require.NoError(t, err)
	defer res.Body.Close()
	// ogen: missing required query parameter rejected at decode time
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestVerify_NoAuthRequired(t *testing.T) {
	// /verify is in the spec security: [] override; should not require a token.
	repo := &mockDocumentRepo{}
	repo.On("FindByHash", "x").Return(nil, gorm.ErrRecordNotFound)

	srv := newTestServer(t, verifyServer(repo))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/verify?hash=x", nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
