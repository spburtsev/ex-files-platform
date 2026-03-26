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

	assignv1 "github.com/spburtsev/ex-files-backend/gen/assignments/v1"
	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockAssignmentRepo struct{ mock.Mock }

func (m *mockAssignmentRepo) ListAll() ([]models.Assignment, error) {
	args := m.Called()
	if v, ok := args.Get(0).([]models.Assignment); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAssignmentRepo) FindByID(id uint) (*models.Assignment, error) {
	args := m.Called(id)
	if v, ok := args.Get(0).(*models.Assignment); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAssignmentRepo) Create(a *models.Assignment) error {
	return m.Called(a).Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func serveAssignment(handler gin.HandlerFunc, method, path, routePattern string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Handle(method, routePattern, handler)
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func newAssignmentsHandler(aRepo *mockAssignmentRepo, uRepo *mockRepo) *handlers.AssignmentsHandler {
	return &handlers.AssignmentsHandler{Repo: aRepo, UserRepo: uRepo}
}

// Test fixtures
var testAssignUser = models.User{Name: "Alex Johnson", Email: "a.johnson@acme.org", Role: models.RoleEmployee}
var testAssignment = models.Assignment{
	Title:       "Sorting Algorithms Report",
	Description: "Implement and benchmark QuickSort.",
	Resolved:    false,
	Assignee:    testAssignUser,
}

func init() {
	testAssignUser.ID = 2
	testAssignment.ID = 1
	testAssignment.CreatorID = 5
	testAssignment.AssigneeID = 2
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

	h := newAssignmentsHandler(&mockAssignmentRepo{}, uRepo)
	w := serveAssignment(h.GetUsers, http.MethodGet, "/users", "/users")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/x-protobuf", w.Header().Get("Content-Type"))

	var resp assignv1.GetUsersResponse
	require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, "Alex Johnson", resp.Users[0].Name)
}

func TestGetUsers_RepoError(t *testing.T) {
	uRepo := &mockRepo{}
	uRepo.On("ListAll").Return(nil, errors.New("db error"))

	h := newAssignmentsHandler(&mockAssignmentRepo{}, uRepo)
	w := serveAssignment(h.GetUsers, http.MethodGet, "/users", "/users")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAssignments(t *testing.T) {
	aRepo := &mockAssignmentRepo{}
	aRepo.On("ListAll").Return([]models.Assignment{
		testAssignment,
		{Title: "Binary Search Trees", Resolved: true, Assignee: testAssignUser},
	}, nil)

	h := newAssignmentsHandler(aRepo, &mockRepo{})
	w := serveAssignment(h.GetAssignments, http.MethodGet, "/assignments", "/assignments")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp assignv1.GetAssignmentsResponse
	require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Assignments, 2)
	assert.Equal(t, "Sorting Algorithms Report", resp.Assignments[0].Title)
}

func TestGetAssignments_RepoError(t *testing.T) {
	aRepo := &mockAssignmentRepo{}
	aRepo.On("ListAll").Return(nil, errors.New("db error"))

	h := newAssignmentsHandler(aRepo, &mockRepo{})
	w := serveAssignment(h.GetAssignments, http.MethodGet, "/assignments", "/assignments")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAssignment(t *testing.T) {
	t.Run("valid id returns assignment with assignee", func(t *testing.T) {
		aRepo := &mockAssignmentRepo{}
		aRepo.On("FindByID", uint(1)).Return(&testAssignment, nil)

		h := newAssignmentsHandler(aRepo, &mockRepo{})
		w := serveAssignment(h.GetAssignment, http.MethodGet, "/assignments/1", "/assignments/:id")

		assert.Equal(t, http.StatusOK, w.Code)

		var resp assignv1.GetAssignmentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "1", resp.Assignment.Id)
		assert.Equal(t, "Sorting Algorithms Report", resp.Assignment.Title)
		assert.Equal(t, "Alex Johnson", resp.User.Name)
	})

	t.Run("non-numeric id returns 400", func(t *testing.T) {
		h := newAssignmentsHandler(&mockAssignmentRepo{}, &mockRepo{})
		w := serveAssignment(h.GetAssignment, http.MethodGet, "/assignments/abc", "/assignments/:id")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		aRepo := &mockAssignmentRepo{}
		aRepo.On("FindByID", uint(999)).Return(nil, errors.New("not found"))

		h := newAssignmentsHandler(aRepo, &mockRepo{})
		w := serveAssignment(h.GetAssignment, http.MethodGet, "/assignments/999", "/assignments/:id")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
