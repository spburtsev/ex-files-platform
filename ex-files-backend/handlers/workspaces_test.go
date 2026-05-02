package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func wsServer(tokens *mockTokens, repo *mockWorkspaceRepo, users *mockUserRepo) *handlers.Server {
	return &handlers.Server{
		UserRepo:      users,
		Tokens:        tokens,
		Hasher:        stubHasher{},
		Audit:         &dummyAudit{},
		WorkspaceRepo: repo,
	}
}

func TestWorkspacesCreate_ManagerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("Create", mock.AnythingOfType("*models.Workspace")).Return(uint(10), nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"name":"Engineering"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode)
	var got oapi.CreateWorkspaceResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "10", got.Workspace.ID)
	assert.Equal(t, "Engineering", got.Workspace.Name)
	assert.Equal(t, "1", got.Workspace.ManagerId)
}

func TestWorkspacesCreate_EmployeeForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)

	srv := newTestServer(t, wsServer(tokens, &mockWorkspaceRepo{}, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"name":"Engineering"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestWorkspacesList_PaginationHeaders(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByManager", uint(1), 20, 0).Return([]models.Workspace{
		{Model: gormModelID(1), Name: "A", ManagerID: 1},
		{Model: gormModelID(2), Name: "B", ManagerID: 1},
	}, int64(42), nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "42", res.Header.Get("X-Total-Count"))
	assert.Equal(t, "1", res.Header.Get("X-Page"))
	assert.Equal(t, "20", res.Header.Get("X-Per-Page"))
	assert.Equal(t, "3", res.Header.Get("X-Total-Pages"))

	var got oapi.GetWorkspacesResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Workspaces, 2)
}

func TestWorkspacesList_EmployeeUsesMemberQuery(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 5, models.RoleEmployee)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByMember", uint(5), 20, 0).Return([]models.Workspace{}, int64(0), nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	repo.AssertCalled(t, "FindByMember", uint(5), 20, 0)
	repo.AssertNotCalled(t, "FindByManager", mock.Anything, mock.Anything, mock.Anything)
}

func TestWorkspacesGet_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	users := &mockUserRepo{}

	repo.On("FindByID", uint(7)).Return(&models.Workspace{Model: gormModelID(7), Name: "Eng", ManagerID: 1}, nil)
	users.On("FindByID", uint(1)).Return(&models.User{Model: gormModelID(1), Email: "m@x", Name: "Man", Role: models.RoleManager}, nil)
	repo.On("GetMembers", uint(7)).Return([]models.User{
		{Model: gormModelID(2), Email: "u@x", Name: "U", Role: models.RoleEmployee},
	}, nil)

	srv := newTestServer(t, wsServer(tokens, repo, users))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/7", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetWorkspaceResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "7", got.Workspace.Workspace.ID)
	assert.Equal(t, "Man", got.Workspace.Manager.Name)
	assert.Len(t, got.Workspace.Members, 1)
}

func TestWorkspacesGet_NotFound(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByID", uint(99)).Return(nil, errors.New("not found"))

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/99", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestWorkspacesUpdate_OwnerOnly(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 2, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByID", uint(7)).Return(&models.Workspace{Model: gormModelID(7), Name: "Old", ManagerID: 99}, nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"name":"New"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/workspaces/7", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestWorkspacesDelete_OwnerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByID", uint(7)).Return(&models.Workspace{Model: gormModelID(7), Name: "X", ManagerID: 1}, nil)
	repo.On("Delete", uint(7)).Return(nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/workspaces/7", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	var msg oapi.MessageResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&msg))
	assert.Equal(t, "workspace deleted", msg.Message)
}

func TestWorkspacesAddMember_OwnerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByID", uint(7)).Return(&models.Workspace{Model: gormModelID(7), ManagerID: 1}, nil)
	repo.On("AddMember", mock.AnythingOfType("*models.WorkspaceMember")).Return(uint(123), nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"userId":"55"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces/7/members", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode)
	var got oapi.AddMemberResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "123", got.Member.ID)
	assert.Equal(t, "55", got.Member.UserId)
	assert.Equal(t, "7", got.Member.WorkspaceId)
}

func TestWorkspacesAddMember_BadUserIdReturns400(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByID", uint(7)).Return(&models.Workspace{Model: gormModelID(7), ManagerID: 1}, nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	body := strings.NewReader(`{"userId":"not-a-number"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/workspaces/7/members", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestWorkspacesRemoveMember_OwnerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("FindByID", uint(7)).Return(&models.Workspace{Model: gormModelID(7), ManagerID: 1}, nil)
	repo.On("RemoveMember", uint(7), uint(2)).Return(nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/workspaces/7/members/2", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestWorkspacesAssignableMembers_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockWorkspaceRepo{}
	repo.On("GetAssignableUsers", uint(7)).Return([]models.User{
		{Model: gormModelID(2), Email: "a@x", Name: "A", Role: models.RoleEmployee},
	}, nil)

	srv := newTestServer(t, wsServer(tokens, repo, &mockUserRepo{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/workspaces/7/assignable-members", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.AssignableMembersResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Users, 1)
}

