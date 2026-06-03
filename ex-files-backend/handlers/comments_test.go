package handlers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

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

	body := strings.NewReader(`{"body":"Nice work","metadata":{"page":1,"x":0.5,"y":0.25}}`)
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

	body := strings.NewReader(`{"body":"","metadata":{"page":1,"x":0,"y":0}}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/comments", body))
	require.NoError(t, err)
	defer res.Body.Close()
	// minLength: 1 in spec causes ogen to validate at decode time → 400
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

// Metadata from datatypes.JSONMap.Scan arrives as json.Number; exercise the
// commentToOAPI -> jsonNumberToFloat conversion path.
func TestCommentsList_WithNumericMetadata(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockCommentRepo{}
	repo.On("ListByDocument", uint(42)).Return([]models.Comment{
		{
			Model:      gormModelID(1),
			DocumentID: 42,
			AuthorID:   1,
			Author:     models.User{Model: gormModelID(1), Name: "Alice"},
			Body:       "pinned",
			Metadata: datatypes.JSONMap{
				"page": json.Number("2"),
				"x":    json.Number("0.5"),
				"y":    json.Number("0.25"),
			},
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
	assert.Equal(t, 2, got.Comments[0].Metadata.Page)
	assert.InEpsilon(t, 0.5, got.Comments[0].Metadata.X, 1e-9)
	assert.InEpsilon(t, 0.25, got.Comments[0].Metadata.Y, 1e-9)
}

func TestCommentsDelete_AuthorSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockCommentRepo{}
	repo.On("FindByID", uint(50)).Return(&models.Comment{Model: gormModelID(50), DocumentID: 42, AuthorID: 1}, nil)
	repo.On("Delete", uint(50)).Return(nil)

	srv := newTestServer(t, commentsServer(tokens, repo))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42/comments/50", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusNoContent, res.StatusCode)
	repo.AssertCalled(t, "Delete", uint(50))
}

func TestCommentsDelete_NonAuthorForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockCommentRepo{}
	repo.On("FindByID", uint(50)).Return(&models.Comment{Model: gormModelID(50), DocumentID: 42, AuthorID: 2}, nil)

	srv := newTestServer(t, commentsServer(tokens, repo))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42/comments/50", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	repo.AssertNotCalled(t, "Delete", mock.Anything)
}

func TestCommentsDelete_NotFound(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockCommentRepo{}
	repo.On("FindByID", uint(50)).Return(nil, assert.AnError)

	srv := newTestServer(t, commentsServer(tokens, repo))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42/comments/50", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestCommentsDelete_Unauthorized(t *testing.T) {
	srv := newTestServer(t, commentsServer(&mockTokens{}, &mockCommentRepo{}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/documents/42/comments/50", nil)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}
