package handlers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

func commentsServer(tokens *mockTokens, repo *mockCommentRepo) *handlers.Server {
	return &handlers.Server{
		UserRepo:    &mockUserRepo{},
		Tokens:      tokens,
		Hasher:      stubHasher{},
		Audit:       &dummyAudit{},
		CommentRepo: repo,
		Hub:         services.NewSSEHub(),
	}
}

func TestCommentsList_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockCommentRepo{}
	repo.On("ListByDocument", uint(42)).Return([]models.Comment{
		{
			Model:      gormModelID(1),
			DocumentID: 42,
			AuthorID:   1,
			Author:     models.User{Model: gormModelID(1), Name: "Alice"},
			Body:       "Looks good",
		},
	}, nil)

	srv := newTestServer(t, commentsServer(tokens, repo))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42/comments", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.ListCommentsResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	require.Len(t, got.Comments, 1)
	assert.Equal(t, "Alice", got.Comments[0].AuthorName)
	assert.Equal(t, "Looks good", got.Comments[0].Body)
	assert.Equal(t, "42", got.Comments[0].DocumentId)
}

func TestCommentsList_RequiresAuth(t *testing.T) {
	srv := newTestServer(t, commentsServer(&mockTokens{}, &mockCommentRepo{}))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/documents/42/comments")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestCommentsCreate_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockCommentRepo{}
	repo.On("Create", mock.AnythingOfType("*models.Comment")).Return(uint(50), nil)
	repo.On("FindByID", uint(50)).Return(&models.Comment{
		Model:      gormModelID(50),
		DocumentID: 42,
		AuthorID:   1,
		Author:     models.User{Model: gormModelID(1), Name: "Alice"},
		Body:       "Nice work",
	}, nil)

	srv := newTestServer(t, commentsServer(tokens, repo))
	defer srv.Close()

	body := strings.NewReader(`{"body":"Nice work"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/comments", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode)
	var got oapi.CreateCommentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "50", got.Comment.ID)
	assert.Equal(t, "Nice work", got.Comment.Body)
	assert.Equal(t, "Alice", got.Comment.AuthorName)
}

func TestCommentsCreate_EmptyBodyReturns400(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)

	srv := newTestServer(t, commentsServer(tokens, &mockCommentRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"body":""}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/comments", body))
	require.NoError(t, err)
	defer res.Body.Close()
	// minLength: 1 in spec causes ogen to validate at decode time → 400
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}
