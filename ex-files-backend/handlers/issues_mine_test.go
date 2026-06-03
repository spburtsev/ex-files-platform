package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func TestIssuesListMine_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 7, models.RoleEmployee)
	issues := &mockIssueRepo{}
	deadline := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	issues.On("ListMyCurrentIssues", uint(7), "", mock.Anything, mock.Anything).Return([]models.Issue{
		{
			Model:       gormModelID(1),
			Title:       "My issue",
			WorkspaceID: 3,
			Workspace: models.Workspace{
				Model:   gormModelID(3),
				Name:    "Acme",
				Manager: models.User{Model: gormModelID(5), Name: "Boss"},
			},
			Deadline: &deadline,
		},
	}, int64(1), nil)

	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: tokens, Hasher: stubHasher{}, IssueRepo: issues}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/issues/mine", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetMyIssuesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	require.Len(t, got.Issues, 1)
	assert.Equal(t, "My issue", got.Issues[0].Title)
	assert.Equal(t, "Acme", got.Issues[0].WorkspaceName)
	assert.Equal(t, "Boss", got.Issues[0].WorkspaceManagerName)
	assert.True(t, got.Issues[0].Deadline.Set)
}

func TestIssuesListMine_Unauthorized(t *testing.T) {
	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: &mockTokens{}, Hasher: stubHasher{}, IssueRepo: &mockIssueRepo{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/issues/mine")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}
