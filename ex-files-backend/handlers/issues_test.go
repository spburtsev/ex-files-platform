package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func issuesServer(tokens *mockTokens, repo *mockIssueRepo, users *mockUserRepo) *handlers.Server {
	return &handlers.Server{
		UserRepo:  users,
		Tokens:    tokens,
		Hasher:    stubHasher{},
		Audit:     &dummyAudit{},
		IssueRepo: repo,
	}
}

func TestIssuesListByWorkspace_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	repo.On("ListByWorkspace", uint(7), "", (*bool)(nil), false).Return([]models.Issue{
		{Model: gormModelID(1), WorkspaceID: 7, CreatorID: 1, AssigneeID: 2, Title: "A"},
		{Model: gormModelID(2), WorkspaceID: 7, CreatorID: 1, AssigneeID: 3, Title: "B"},
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/7/issues", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssuesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Issues, 2)
	assert.Equal(t, "A", got.Issues[0].Title)
	assert.Equal(t, "7", got.Issues[0].WorkspaceId)
}

func TestIssuesListByWorkspace_RequiresAuth(t *testing.T) {
	srv := newTestServer(t, issuesServer(&mockTokens{}, &mockIssueRepo{}, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/workspaces/7/issues")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestIssuesGet_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockIssueRepo{}
	deadline := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model:       gormModelID(42),
		WorkspaceID: 7,
		CreatorID:   1,
		AssigneeID:  2,
		Assignee:    models.User{Model: gormModelID(2), Email: "a@x", Name: "Assignee", Role: models.RoleEmployee},
		Title:       "Review",
		Description: "Quarterly review",
		Deadline:    &deadline,
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/issues/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssueResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "42", got.Issue.ID)
	assert.Equal(t, "Review", got.Issue.Title)
	require.True(t, got.Issue.Deadline.IsSet())
	dl, _ := got.Issue.Deadline.Get()
	assert.Equal(t, deadline.Unix(), dl.Unix())
	assert.Equal(t, "Assignee", got.User.Name)
}

func TestIssuesGet_NotFound(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockIssueRepo{}
	repo.On("FindByID", uint(999)).Return(nil, errors.New("not found"))

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/issues/999", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestIssuesCreate_ManagerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	repo.On("Create", mock.AnythingOfType("*models.Issue")).Return(uint(11), nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"title":"Q4 audit","description":"Stuff","assigneeId":"2","deadline":"2026-12-31T23:59:59Z"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces/7/issues", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode)
	var got oapi.CreateIssueResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "11", got.Issue.ID)
	assert.Equal(t, "Q4 audit", got.Issue.Title)
}

func TestIssuesCreate_EmployeeForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)

	srv := newTestServer(t, issuesServer(tokens, &mockIssueRepo{}, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"title":"X","assigneeId":"2"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces/7/issues", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestIssuesCreate_BadAssigneeReturns400(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)

	srv := newTestServer(t, issuesServer(tokens, &mockIssueRepo{}, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"title":"X","assigneeId":"not-a-number"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces/7/issues", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestIssuesListByWorkspace_StatusOpen(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	f := false
	repo.On("ListByWorkspace", uint(7), "", &f, false).Return([]models.Issue{
		{Model: gormModelID(1), WorkspaceID: 7, Title: "Open issue"},
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/7/issues?status=open", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssuesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Issues, 1)
}

func TestIssuesListByWorkspace_StatusResolved(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	tr := true
	repo.On("ListByWorkspace", uint(7), "", &tr, false).Return([]models.Issue{
		{Model: gormModelID(2), WorkspaceID: 7, Title: "Done", Resolved: true},
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/7/issues?status=resolved", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssuesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Issues, 1)
	assert.True(t, got.Issues[0].Resolved)
}

func TestIssuesListByWorkspace_SearchParam(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	repo.On("ListByWorkspace", uint(7), "foo", (*bool)(nil), false).Return([]models.Issue{
		{Model: gormModelID(3), WorkspaceID: 7, Title: "foo bar"},
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/7/issues?search=foo", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssuesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Issues, 1)
}

func TestIssuesUpdateAssignee_ManagerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	users := &mockUserRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model:       gormModelID(42),
		WorkspaceID: 7,
		CreatorID:   99,
		AssigneeID:  2,
		Title:       "Review",
	}, nil).Once()
	users.On("FindByID", uint(5)).Return(&models.User{Model: gormModelID(5), Name: "New Assignee"}, nil)
	repo.On("Update", mock.MatchedBy(func(i *models.Issue) bool { return i.AssigneeID == 5 })).Return(nil)
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model:       gormModelID(42),
		WorkspaceID: 7,
		CreatorID:   99,
		AssigneeID:  5,
		Assignee:    models.User{Model: gormModelID(5), Name: "New Assignee"},
		Title:       "Review",
	}, nil).Once()

	srv := newTestServer(t, issuesServer(tokens, repo, users))
	defer srv.Close()

	body := strings.NewReader(`{"assigneeId":"5"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/42/assignee", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssueResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "5", got.Issue.AssigneeId)
	assert.Equal(t, "New Assignee", got.User.Name)
}

func TestIssuesUpdateAssignee_CreatorSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleEmployee)
	repo := &mockIssueRepo{}
	users := &mockUserRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model:      gormModelID(42),
		CreatorID:  99,
		AssigneeID: 2,
	}, nil).Once()
	users.On("FindByID", uint(5)).Return(&models.User{Model: gormModelID(5), Name: "New"}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil)
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model:      gormModelID(42),
		CreatorID:  99,
		AssigneeID: 5,
		Assignee:   models.User{Model: gormModelID(5), Name: "New"},
	}, nil).Once()

	srv := newTestServer(t, issuesServer(tokens, repo, users))
	defer srv.Close()

	body := strings.NewReader(`{"assigneeId":"5"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/42/assignee", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestIssuesUpdateAssignee_NonCreatorEmployeeForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 7, models.RoleEmployee)
	repo := &mockIssueRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model:     gormModelID(42),
		CreatorID: 99,
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"assigneeId":"5"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/42/assignee", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestIssuesUpdateAssignee_ResolvedReturns422(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model:    gormModelID(42),
		Resolved: true,
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"assigneeId":"5"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/42/assignee", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
}

func TestIssuesUpdateAssignee_UnknownAssignee400(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	users := &mockUserRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Issue{Model: gormModelID(42)}, nil)
	users.On("FindByID", uint(999)).Return(nil, errors.New("not found"))

	srv := newTestServer(t, issuesServer(tokens, repo, users))
	defer srv.Close()

	body := strings.NewReader(`{"assigneeId":"999"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/42/assignee", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestIssuesListByWorkspace_ArchivedParam(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	repo.On("ListByWorkspace", uint(7), "", (*bool)(nil), true).Return([]models.Issue{
		{Model: gormModelID(5), WorkspaceID: 7, Title: "Archived issue", Archived: true},
	}, nil)

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/7/issues?archived=true", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssuesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Issues, 1)
}

func TestIssuesArchive_ManagerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockIssueRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model: gormModelID(42), WorkspaceID: 7, Title: "Review",
		Assignee: models.User{Model: gormModelID(2), Name: "A"},
	}, nil).Once()
	repo.On("Update", mock.MatchedBy(func(i *models.Issue) bool { return i.Archived })).Return(nil)
	repo.On("FindByID", uint(42)).Return(&models.Issue{
		Model: gormModelID(42), WorkspaceID: 7, Title: "Review", Archived: true,
		Assignee: models.User{Model: gormModelID(2), Name: "A"},
	}, nil).Once()

	srv := newTestServer(t, issuesServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"archived":true}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/42/archive", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetIssueResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	archived, ok := got.Issue.Archived.Get()
	require.True(t, ok)
	assert.True(t, archived)
}

func TestIssuesArchive_EmployeeForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)

	srv := newTestServer(t, issuesServer(tokens, &mockIssueRepo{}, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"archived":true}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/42/archive", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}
