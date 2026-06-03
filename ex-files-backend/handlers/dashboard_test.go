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
	"github.com/spburtsev/ex-files-backend/services"
)

func TestDashboardGet_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 7, models.RoleManager)
	issues := &mockIssueRepo{}
	activity := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	issues.On("DashboardSummary", uint(7), mock.Anything).Return(services.DashboardSummary{
		AssignedOpenCount: 2,
		CreatedOpenCount:  1,
		DueSoon:           []models.Issue{{Model: gormModelID(1), Title: "Due soon", CreatorID: 7, AssigneeID: 7, WorkspaceID: 1}},
		Overdue:           []models.Issue{{Model: gormModelID(2), Title: "Overdue", CreatorID: 7, AssigneeID: 7, WorkspaceID: 1}},
		Recent: []services.IssueWithActivity{
			{Issue: models.Issue{Model: gormModelID(3), Title: "Recent", CreatorID: 7, AssigneeID: 7, WorkspaceID: 1}, LastActivityAt: activity},
		},
	}, nil)

	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: tokens, Hasher: stubHasher{}, IssueRepo: issues}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/dashboard", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.DashboardResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, int64(2), got.AssignedOpenCount)
	assert.Equal(t, int64(1), got.CreatedOpenCount)
	require.Len(t, got.DueSoon, 1)
	require.Len(t, got.Overdue, 1)
	require.Len(t, got.Recent, 1)
	assert.True(t, got.Recent[0].LastActivityAt.Set, "recent item carries LastActivityAt")
}

func TestDashboardGet_Unauthorized(t *testing.T) {
	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: &mockTokens{}, Hasher: stubHasher{}, IssueRepo: &mockIssueRepo{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/dashboard")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestDashboardGet_SummaryErrorReturns500(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 7, models.RoleManager)
	issues := &mockIssueRepo{}
	issues.On("DashboardSummary", uint(7), mock.Anything).Return(services.DashboardSummary{}, assert.AnError)

	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: tokens, Hasher: stubHasher{}, IssueRepo: issues}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/dashboard", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}
