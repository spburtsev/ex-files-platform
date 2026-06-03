package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

func reviewConfigServer(tokens *mockTokens, issues *mockIssueRepo, ws *mockWorkspaceRepo) *handlers.Server {
	return &handlers.Server{
		UserRepo:      &mockUserRepo{},
		Tokens:        tokens,
		Hasher:        stubHasher{},
		IssueRepo:     issues,
		WorkspaceRepo: ws,
	}
}

func configIssue(creatorID uint, resolved bool) *models.Issue {
	return &models.Issue{Model: gormModelID(7), WorkspaceID: 3, CreatorID: creatorID, AssigneeID: 1, Resolved: resolved}
}

func TestIssuesUpdateReviewConfig_CreatorAllowed(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager) // the issue creator
	issues := &mockIssueRepo{}
	ws := &mockWorkspaceRepo{}

	issues.On("FindByID", uint(7)).Return(configIssue(99, false), nil)
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 1}, nil)
	ws.On("GetMembers", uint(3)).Return(usersWithIDs(5, 6, 7), nil)
	var gotIDs []uint
	issues.On("SetReviewers", uint(7), mock.Anything).Return(nil).Run(func(a mock.Arguments) {
		gotIDs = a.Get(1).([]uint)
	})
	issues.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil)

	srv := newTestServer(t, reviewConfigServer(tokens, issues, ws))
	defer srv.Close()

	body := strings.NewReader(`{"reviewerIds":["5","6"],"requiredApprovals":2}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/7/review-config", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, []uint{5, 6}, gotIDs)
}

func TestIssuesUpdateReviewConfig_OwnerAllowed(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager) // workspace owner, not the creator
	issues := &mockIssueRepo{}
	ws := &mockWorkspaceRepo{}

	issues.On("FindByID", uint(7)).Return(configIssue(99, false), nil)
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 1}, nil)
	ws.On("GetMembers", uint(3)).Return(usersWithIDs(5, 6, 7), nil)
	issues.On("SetReviewers", uint(7), mock.Anything).Return(nil)
	issues.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil)

	srv := newTestServer(t, reviewConfigServer(tokens, issues, ws))
	defer srv.Close()

	body := strings.NewReader(`{"reviewerIds":["5"],"requiredApprovals":1}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/7/review-config", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
}

func TestIssuesUpdateReviewConfig_OtherManagerForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 2, models.RoleManager) // a manager who is neither creator nor owner
	issues := &mockIssueRepo{}
	ws := &mockWorkspaceRepo{}

	issues.On("FindByID", uint(7)).Return(configIssue(99, false), nil)
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 1}, nil)

	srv := newTestServer(t, reviewConfigServer(tokens, issues, ws))
	defer srv.Close()

	body := strings.NewReader(`{"reviewerIds":["5"],"requiredApprovals":1}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/7/review-config", body))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	issues.AssertNotCalled(t, "SetReviewers", mock.Anything, mock.Anything)
}

func TestIssuesUpdateReviewConfig_NExceedsPanelReturns422(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	issues := &mockIssueRepo{}
	ws := &mockWorkspaceRepo{}

	issues.On("FindByID", uint(7)).Return(configIssue(99, false), nil)
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 1}, nil)
	ws.On("GetMembers", uint(3)).Return(usersWithIDs(5, 6, 7), nil)

	srv := newTestServer(t, reviewConfigServer(tokens, issues, ws))
	defer srv.Close()

	body := strings.NewReader(`{"reviewerIds":["5","6"],"requiredApprovals":3}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/7/review-config", body))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
	issues.AssertNotCalled(t, "SetReviewers", mock.Anything, mock.Anything)
}

func TestIssuesUpdateReviewConfig_ResolvedIssueReturns422(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	issues := &mockIssueRepo{}
	ws := &mockWorkspaceRepo{}

	issues.On("FindByID", uint(7)).Return(configIssue(99, true), nil)
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 1}, nil)

	srv := newTestServer(t, reviewConfigServer(tokens, issues, ws))
	defer srv.Close()

	body := strings.NewReader(`{"reviewerIds":["5"],"requiredApprovals":1}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/issues/7/review-config", body))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
}

func TestIssuesCreate_WithReviewersClampsN(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	issues := &mockIssueRepo{}
	ws := &mockWorkspaceRepo{}

	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 1}, nil)
	ws.On("GetMembers", uint(3)).Return(usersWithIDs(5, 6, 7), nil)
	var createdN int
	issues.On("Create", mock.AnythingOfType("*models.Issue")).Return(uint(7), nil).Run(func(a mock.Arguments) {
		createdN = a.Get(0).(*models.Issue).RequiredApprovals
	})
	issues.On("SetReviewers", uint(7), mock.Anything).Return(nil)
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3, CreatorID: 1, AssigneeID: 2}, nil)

	srv := newTestServer(t, &handlers.Server{
		UserRepo: &mockUserRepo{}, Tokens: tokens, Hasher: stubHasher{},
		IssueRepo: issues, WorkspaceRepo: ws,
	})
	defer srv.Close()

	// requiredApprovals 5 but only 2 reviewers -> clamped to 2.
	body := strings.NewReader(`{"title":"X","assigneeId":"2","reviewerIds":["5","6"],"requiredApprovals":5}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces/3/issues", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Equal(t, 2, createdN, "requiredApprovals must be clamped to the panel size")
}
