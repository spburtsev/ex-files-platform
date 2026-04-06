package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	workspacesv1 "github.com/spburtsev/ex-files-backend/gen/workspaces/v1"
	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

// --- Workspace mock repo ---

type mockWSRepo struct{ mock.Mock }

func (m *mockWSRepo) Create(ws *models.Workspace) error {
	args := m.Called(ws)
	ws.ID = 1
	return args.Error(0)
}

func (m *mockWSRepo) FindByID(id uint) (*models.Workspace, error) {
	args := m.Called(id)
	if ws, ok := args.Get(0).(*models.Workspace); ok {
		return ws, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockWSRepo) FindByManager(managerID uint, limit, offset int) ([]models.Workspace, int64, error) {
	args := m.Called(managerID, limit, offset)
	return args.Get(0).([]models.Workspace), args.Get(1).(int64), args.Error(2)
}

func (m *mockWSRepo) FindByMember(userID uint, limit, offset int) ([]models.Workspace, int64, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]models.Workspace), args.Get(1).(int64), args.Error(2)
}

func (m *mockWSRepo) Update(ws *models.Workspace) error {
	return m.Called(ws).Error(0)
}

func (m *mockWSRepo) Delete(id uint) error {
	return m.Called(id).Error(0)
}

func (m *mockWSRepo) AddMember(member *models.WorkspaceMember) error {
	args := m.Called(member)
	member.ID = 1
	return args.Error(0)
}

func (m *mockWSRepo) RemoveMember(workspaceID, userID uint) error {
	return m.Called(workspaceID, userID).Error(0)
}

func (m *mockWSRepo) GetMembers(workspaceID uint) ([]models.User, error) {
	args := m.Called(workspaceID)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *mockWSRepo) GetAssignableUsers(workspaceID uint) ([]models.User, error) {
	args := m.Called(workspaceID)
	if u, ok := args.Get(0).([]models.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

// --- helpers ---

func newWSHandler(wsRepo *mockWSRepo, userRepo *mockRepo) *handlers.WorkspaceHandler {
	return &handlers.WorkspaceHandler{Repo: wsRepo, UserRepo: userRepo}
}

func wsRequest(h gin.HandlerFunc, method, path, routePattern string, body *bytes.Buffer, userID uint, role string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Handle(method, routePattern, func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", role)
		h(c)
	})
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	r.ServeHTTP(w, req)
	return w
}

// --- TestCreate ---

func TestWorkspaceCreate(t *testing.T) {
	t.Run("forbidden_for_employee", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]string{"name": "Test WS"})
		w := wsRequest(h.Create, http.MethodPost, "/workspaces", "/workspaces", body, 1, "employee")

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("bad_json", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		h := newWSHandler(wsRepo, &mockRepo{})
		body := bytes.NewBufferString("not-json")
		w := wsRequest(h.Create, http.MethodPost, "/workspaces", "/workspaces", body, 1, "manager")

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("db_failure", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("Create", mock.AnythingOfType("*models.Workspace")).Return(errors.New("db error"))
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]string{"name": "Test WS"})
		w := wsRequest(h.Create, http.MethodPost, "/workspaces", "/workspaces", body, 1, "manager")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("success_manager", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("Create", mock.AnythingOfType("*models.Workspace")).Return(nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]string{"name": "Test WS"})
		w := wsRequest(h.Create, http.MethodPost, "/workspaces", "/workspaces", body, 5, "manager")

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/x-protobuf", w.Header().Get("Content-Type"))
		var resp workspacesv1.CreateWorkspaceResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Test WS", resp.Workspace.Name)
		wsRepo.AssertExpectations(t)
	})

	t.Run("success_root", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("Create", mock.AnythingOfType("*models.Workspace")).Return(nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]string{"name": "Root WS"})
		w := wsRequest(h.Create, http.MethodPost, "/workspaces", "/workspaces", body, 1, "root")

		assert.Equal(t, http.StatusCreated, w.Code)
		wsRepo.AssertExpectations(t)
	})
}

// --- TestList ---

func TestWorkspaceList(t *testing.T) {
	t.Run("manager_sees_owned", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("FindByManager", uint(5), 20, 0).Return(
			[]models.Workspace{{Name: "WS1"}}, int64(1), nil,
		)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.List, http.MethodGet, "/workspaces", "/workspaces", nil, 5, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "1", w.Header().Get("X-Total-Count"))
		assert.Equal(t, "1", w.Header().Get("X-Page"))
		wsRepo.AssertExpectations(t)
	})

	t.Run("employee_sees_memberships", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("FindByMember", uint(2), 20, 0).Return(
			[]models.Workspace{{Name: "WS1"}, {Name: "WS2"}}, int64(2), nil,
		)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.List, http.MethodGet, "/workspaces", "/workspaces", nil, 2, "employee")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "2", w.Header().Get("X-Total-Count"))
		wsRepo.AssertExpectations(t)
	})

	t.Run("pagination_params", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("FindByManager", uint(5), 10, 10).Return(
			[]models.Workspace{}, int64(15), nil,
		)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.List, http.MethodGet, "/workspaces?page=2&per_page=10", "/workspaces", nil, 5, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "15", w.Header().Get("X-Total-Count"))
		assert.Equal(t, "2", w.Header().Get("X-Page"))
		assert.Equal(t, "10", w.Header().Get("X-Per-Page"))
		assert.Equal(t, "2", w.Header().Get("X-Total-Pages"))
		wsRepo.AssertExpectations(t)
	})

	t.Run("db_failure", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("FindByManager", uint(5), 20, 0).Return(
			[]models.Workspace(nil), int64(0), errors.New("db error"),
		)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.List, http.MethodGet, "/workspaces", "/workspaces", nil, 5, "manager")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- TestGet ---

func TestWorkspaceGet(t *testing.T) {
	t.Run("invalid_id", func(t *testing.T) {
		h := newWSHandler(&mockWSRepo{}, &mockRepo{})
		w := wsRequest(h.Get, http.MethodGet, "/workspaces/abc", "/workspaces/:id", nil, 1, "manager")

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not_found", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		wsRepo.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.Get, http.MethodGet, "/workspaces/99", "/workspaces/:id", nil, 1, "manager")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		userRepo := &mockRepo{}
		ws := &models.Workspace{Name: "My WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		userRepo.On("FindByID", uint(5)).Return(&models.User{Email: "mgr@test.com", Name: "Manager", Role: models.RoleManager}, nil)
		wsRepo.On("GetMembers", uint(1)).Return([]models.User{
			{Email: "emp@test.com", Name: "Employee", Role: models.RoleEmployee},
		}, nil)
		h := newWSHandler(wsRepo, userRepo)
		w := wsRequest(h.Get, http.MethodGet, "/workspaces/1", "/workspaces/:id", nil, 5, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		var resp workspacesv1.GetWorkspaceResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Manager", resp.Workspace.Manager.Name)
		assert.Len(t, resp.Workspace.Members, 1)
		wsRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})
}

// --- TestUpdate ---

func TestWorkspaceUpdate(t *testing.T) {
	t.Run("not_owner", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]string{"name": "New Name"})
		w := wsRequest(h.Update, http.MethodPut, "/workspaces/1", "/workspaces/:id", body, 99, "manager")

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		wsRepo.On("Update", ws).Return(nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]string{"name": "Updated"})
		w := wsRequest(h.Update, http.MethodPut, "/workspaces/1", "/workspaces/:id", body, 5, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		wsRepo.AssertExpectations(t)
	})
}

// --- TestDelete ---

func TestWorkspaceDelete(t *testing.T) {
	t.Run("not_owner", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.Delete, http.MethodDelete, "/workspaces/1", "/workspaces/:id", nil, 99, "manager")

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		wsRepo.On("Delete", uint(1)).Return(nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.Delete, http.MethodDelete, "/workspaces/1", "/workspaces/:id", nil, 5, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		wsRepo.AssertExpectations(t)
	})
}

// --- TestAddMember ---

func TestWorkspaceAddMember(t *testing.T) {
	t.Run("not_owner", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]any{"user_id": 2})
		w := wsRequest(h.AddMember, http.MethodPost, "/workspaces/1/members", "/workspaces/:id/members", body, 99, "manager")

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		wsRepo.On("AddMember", mock.AnythingOfType("*models.WorkspaceMember")).Return(nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		body := jsonBody(t, map[string]any{"user_id": 2})
		w := wsRequest(h.AddMember, http.MethodPost, "/workspaces/1/members", "/workspaces/:id/members", body, 5, "manager")

		assert.Equal(t, http.StatusCreated, w.Code)
		wsRepo.AssertExpectations(t)
	})
}

// --- TestRemoveMember ---

func TestWorkspaceRemoveMember(t *testing.T) {
	t.Run("not_owner", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.RemoveMember, http.MethodDelete, "/workspaces/1/members/2", "/workspaces/:id/members/:userId", nil, 99, "manager")

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		wsRepo.On("RemoveMember", uint(1), uint(2)).Return(nil)
		h := newWSHandler(wsRepo, &mockRepo{})
		w := wsRequest(h.RemoveMember, http.MethodDelete, "/workspaces/1/members/2", "/workspaces/:id/members/:userId", nil, 5, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		wsRepo.AssertExpectations(t)
	})
}
