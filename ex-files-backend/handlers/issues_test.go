package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	issuesv1 "github.com/spburtsev/ex-files-backend/gen/issues/v1"
	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockIssueRepo struct{ mock.Mock }

func (m *mockIssueRepo) ListAll() ([]models.Issue, error) {
	args := m.Called()
	if v, ok := args.Get(0).([]models.Issue); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockIssueRepo) ListByWorkspace(workspaceID uint) ([]models.Issue, error) {
	args := m.Called(workspaceID)
	if v, ok := args.Get(0).([]models.Issue); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockIssueRepo) FindByID(id uint) (*models.Issue, error) {
	args := m.Called(id)
	if v, ok := args.Get(0).(*models.Issue); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockIssueRepo) Create(a *models.Issue) error {
	return m.Called(a).Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func serveIssue(handler gin.HandlerFunc, method, path, routePattern string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Handle(method, routePattern, handler)
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func newIssuesHandler(iRepo *mockIssueRepo, uRepo *mockRepo) *handlers.IssuesHandler {
	return &handlers.IssuesHandler{Repo: iRepo, UserRepo: uRepo}
}

// Test fixtures
var testIssueUser = models.User{Name: "Alex Johnson", Email: "a.johnson@acme.org", Role: models.RoleEmployee}
var testIssue = models.Issue{
	Title:       "Sorting Algorithms Report",
	Description: "Implement and benchmark QuickSort.",
	Resolved:    false,
	Assignee:    testIssueUser,
}

func init() {
	testIssueUser.ID = 2
	testIssue.ID = 1
	testIssue.WorkspaceID = 1
	testIssue.CreatorID = 5
	testIssue.AssigneeID = 2
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestGetUsers(t *testing.T) {
	uRepo := &mockRepo{}
	uRepo.On("ListAll").Return([]models.User{
		{Name: "Alex Johnson", Email: "a.johnson@acme.org", Role: models.RoleEmployee},
		{Name: "Maria Chen", Email: "m.chen@acme.org", Role: models.RoleEmployee},
	}, nil)

	h := newIssuesHandler(&mockIssueRepo{}, uRepo)
	w := serveIssue(h.GetUsers, http.MethodGet, "/users", "/users")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-protobuf", w.Header().Get("Content-Type"))

	var resp issuesv1.GetUsersResponse
	require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, "Alex Johnson", resp.Users[0].Name)
}

func TestGetUsers_RepoError(t *testing.T) {
	uRepo := &mockRepo{}
	uRepo.On("ListAll").Return(nil, errors.New("db error"))

	h := newIssuesHandler(&mockIssueRepo{}, uRepo)
	w := serveIssue(h.GetUsers, http.MethodGet, "/users", "/users")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetIssues(t *testing.T) {
	iRepo := &mockIssueRepo{}
	iRepo.On("ListByWorkspace", uint(1)).Return([]models.Issue{
		testIssue,
		{Title: "Binary Search Trees", Resolved: true, Assignee: testIssueUser},
	}, nil)

	h := newIssuesHandler(iRepo, &mockRepo{})
	w := serveIssue(h.ListByWorkspace, http.MethodGet, "/workspaces/1/issues", "/workspaces/:id/issues")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp issuesv1.GetIssuesResponse
	require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Issues, 2)
	assert.Equal(t, "Sorting Algorithms Report", resp.Issues[0].Title)
}

func TestGetIssues_RepoError(t *testing.T) {
	iRepo := &mockIssueRepo{}
	iRepo.On("ListByWorkspace", uint(1)).Return(nil, errors.New("db error"))

	h := newIssuesHandler(iRepo, &mockRepo{})
	w := serveIssue(h.ListByWorkspace, http.MethodGet, "/workspaces/1/issues", "/workspaces/:id/issues")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetIssue(t *testing.T) {
	t.Run("valid id returns issue with assignee", func(t *testing.T) {
		iRepo := &mockIssueRepo{}
		iRepo.On("FindByID", uint(1)).Return(&testIssue, nil)

		h := newIssuesHandler(iRepo, &mockRepo{})
		w := serveIssue(h.Get, http.MethodGet, "/issues/1", "/issues/:id")

		assert.Equal(t, http.StatusOK, w.Code)

		var resp issuesv1.GetIssueResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "1", resp.Issue.Id)
		assert.Equal(t, "Sorting Algorithms Report", resp.Issue.Title)
		assert.Equal(t, "Alex Johnson", resp.User.Name)
	})

	t.Run("non-numeric id returns 400", func(t *testing.T) {
		h := newIssuesHandler(&mockIssueRepo{}, &mockRepo{})
		w := serveIssue(h.Get, http.MethodGet, "/issues/abc", "/issues/:id")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		iRepo := &mockIssueRepo{}
		iRepo.On("FindByID", uint(999)).Return(nil, errors.New("not found"))

		h := newIssuesHandler(iRepo, &mockRepo{})
		w := serveIssue(h.Get, http.MethodGet, "/issues/999", "/issues/:id")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
