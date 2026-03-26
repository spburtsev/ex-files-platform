package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	auditv1 "github.com/spburtsev/ex-files-backend/gen/audit/v1"
	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

// --- Mock audit repo ---

type mockAuditRepo struct{ mock.Mock }

func (m *mockAuditRepo) Append(entry *models.AuditEntry) error {
	args := m.Called(entry)
	entry.ID = 1
	return args.Error(0)
}

func (m *mockAuditRepo) List(filter services.AuditFilter, limit, offset int) ([]models.AuditEntry, int64, error) {
	args := m.Called(filter, limit, offset)
	return args.Get(0).([]models.AuditEntry), args.Get(1).(int64), args.Error(2)
}

// --- helpers ---

func auditRequest(h gin.HandlerFunc, path string, userID uint, role string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/audit", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", role)
		h(c)
	})
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// --- TestAuditList ---

func TestAuditList(t *testing.T) {
	now := time.Now()

	t.Run("success_no_filters", func(t *testing.T) {
		auditRepo := &mockAuditRepo{}
		entries := []models.AuditEntry{
			{
				ID:        1,
				Action:    models.AuditActionUserRegistered,
				ActorID:   1,
				Actor:     models.User{Name: "Alice"},
				CreatedAt: now,
				Metadata:  datatypes.JSONMap{"email": "alice@test.com"},
			},
			{
				ID:        2,
				Action:    models.AuditActionWorkspaceCreated,
				ActorID:   5,
				Actor:     models.User{Name: "Manager"},
				CreatedAt: now,
				Metadata:  datatypes.JSONMap{"name": "My WS"},
			},
		}
		auditRepo.On("List", mock.AnythingOfType("services.AuditFilter"), 20, 0).Return(entries, int64(2), nil)

		h := &handlers.AuditHandler{Repo: auditRepo}
		w := auditRequest(h.List, "/audit", 1, "root")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "2", w.Header().Get("X-Total-Count"))

		var resp auditv1.GetAuditLogResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp.Entries, 2)

		first := resp.Entries[0]
		assert.Equal(t, "user.registered", first.Action)
		assert.Equal(t, "Alice", first.ActorName)

		auditRepo.AssertExpectations(t)
	})

	t.Run("with_action_filter", func(t *testing.T) {
		auditRepo := &mockAuditRepo{}
		auditRepo.On("List", mock.MatchedBy(func(f services.AuditFilter) bool {
			return f.Action == "workspace.created"
		}), 20, 0).Return([]models.AuditEntry{}, int64(0), nil)

		h := &handlers.AuditHandler{Repo: auditRepo}
		w := auditRequest(h.List, "/audit?action=workspace.created", 1, "root")

		assert.Equal(t, http.StatusOK, w.Code)
		auditRepo.AssertExpectations(t)
	})

	t.Run("with_actor_id_filter", func(t *testing.T) {
		auditRepo := &mockAuditRepo{}
		uid := uint(5)
		auditRepo.On("List", mock.MatchedBy(func(f services.AuditFilter) bool {
			return f.ActorID != nil && *f.ActorID == uid
		}), 20, 0).Return([]models.AuditEntry{}, int64(0), nil)

		h := &handlers.AuditHandler{Repo: auditRepo}
		w := auditRequest(h.List, "/audit?actor_id=5", 1, "root")

		assert.Equal(t, http.StatusOK, w.Code)
		auditRepo.AssertExpectations(t)
	})

	t.Run("with_date_range", func(t *testing.T) {
		auditRepo := &mockAuditRepo{}
		auditRepo.On("List", mock.MatchedBy(func(f services.AuditFilter) bool {
			return f.From != nil && f.To != nil
		}), 20, 0).Return([]models.AuditEntry{}, int64(0), nil)

		h := &handlers.AuditHandler{Repo: auditRepo}
		w := auditRequest(h.List, "/audit?from=2026-01-01T00:00:00Z&to=2026-12-31T23:59:59Z", 1, "root")

		assert.Equal(t, http.StatusOK, w.Code)
		auditRepo.AssertExpectations(t)
	})

	t.Run("pagination", func(t *testing.T) {
		auditRepo := &mockAuditRepo{}
		auditRepo.On("List", mock.AnythingOfType("services.AuditFilter"), 5, 5).Return(
			[]models.AuditEntry{}, int64(12), nil,
		)

		h := &handlers.AuditHandler{Repo: auditRepo}
		w := auditRequest(h.List, "/audit?page=2&per_page=5", 1, "root")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "12", w.Header().Get("X-Total-Count"))
		assert.Equal(t, "2", w.Header().Get("X-Page"))
		assert.Equal(t, "5", w.Header().Get("X-Per-Page"))
		assert.Equal(t, "3", w.Header().Get("X-Total-Pages"))
		auditRepo.AssertExpectations(t)
	})

	t.Run("db_failure", func(t *testing.T) {
		auditRepo := &mockAuditRepo{}
		auditRepo.On("List", mock.AnythingOfType("services.AuditFilter"), 20, 0).Return(
			[]models.AuditEntry(nil), int64(0), errors.New("db error"),
		)

		h := &handlers.AuditHandler{Repo: auditRepo}
		w := auditRequest(h.List, "/audit", 1, "root")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- TestAuditIntegration: verify that handlers call audit logging ---

func TestAuthAuditIntegration(t *testing.T) {
	t.Run("register_logs_audit", func(t *testing.T) {
		repo := &mockRepo{}
		tokens := &mockTokens{}
		hasher := &mockHasher{}
		auditRepo := &mockAuditRepo{}

		repo.On("FindByEmail", "a@b.com").Return(nil, gorm.ErrRecordNotFound)
		hasher.On("Hash", "password1").Return("hashed", nil)
		repo.On("Create", mock.AnythingOfType("*models.User")).Return(nil)
		tokens.On("Issue", mock.AnythingOfType("*models.User")).Return("tok123", nil)
		auditRepo.On("Append", mock.MatchedBy(func(e *models.AuditEntry) bool {
			return e.Action == models.AuditActionUserRegistered
		})).Return(nil)

		h := &handlers.AuthHandler{Repo: repo, Tokens: tokens, Hasher: hasher, Audit: auditRepo}
		body := jsonBody(t, map[string]string{"email": "a@b.com", "password": "password1", "name": "Alice"})
		w := executeRequest(h.Register, http.MethodPost, "/auth/register", body)

		assert.Equal(t, http.StatusCreated, w.Code)
		auditRepo.AssertExpectations(t)
	})

	t.Run("login_logs_audit", func(t *testing.T) {
		repo := &mockRepo{}
		tokens := &mockTokens{}
		hasher := &mockHasher{}
		auditRepo := &mockAuditRepo{}

		u := &models.User{Email: "a@b.com", PasswordHash: "hash"}
		repo.On("FindByEmail", "a@b.com").Return(u, nil)
		hasher.On("Compare", "hash", "password1").Return(nil)
		tokens.On("Issue", u).Return("tok456", nil)
		auditRepo.On("Append", mock.MatchedBy(func(e *models.AuditEntry) bool {
			return e.Action == models.AuditActionUserLoggedIn
		})).Return(nil)

		h := &handlers.AuthHandler{Repo: repo, Tokens: tokens, Hasher: hasher, Audit: auditRepo}
		body := jsonBody(t, map[string]string{"email": "a@b.com", "password": "password1"})
		w := executeRequest(h.Login, http.MethodPost, "/auth/login", body)

		assert.Equal(t, http.StatusOK, w.Code)
		auditRepo.AssertExpectations(t)
	})
}

func TestWorkspaceAuditIntegration(t *testing.T) {
	t.Run("create_logs_audit", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		auditRepo := &mockAuditRepo{}
		wsRepo.On("Create", mock.AnythingOfType("*models.Workspace")).Return(nil)
		auditRepo.On("Append", mock.MatchedBy(func(e *models.AuditEntry) bool {
			return e.Action == models.AuditActionWorkspaceCreated
		})).Return(nil)

		h := &handlers.WorkspaceHandler{Repo: wsRepo, UserRepo: &mockRepo{}, Audit: auditRepo}
		body := jsonBody(t, map[string]string{"name": "WS"})
		w := wsRequest(h.Create, http.MethodPost, "/workspaces", "/workspaces", body, 5, "manager")

		assert.Equal(t, http.StatusCreated, w.Code)
		auditRepo.AssertExpectations(t)
	})

	t.Run("delete_logs_audit", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		auditRepo := &mockAuditRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		wsRepo.On("Delete", uint(1)).Return(nil)
		auditRepo.On("Append", mock.MatchedBy(func(e *models.AuditEntry) bool {
			return e.Action == models.AuditActionWorkspaceDeleted
		})).Return(nil)

		h := &handlers.WorkspaceHandler{Repo: wsRepo, UserRepo: &mockRepo{}, Audit: auditRepo}
		w := wsRequest(h.Delete, http.MethodDelete, "/workspaces/1", "/workspaces/:id", nil, 5, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		auditRepo.AssertExpectations(t)
	})

	t.Run("add_member_logs_audit", func(t *testing.T) {
		wsRepo := &mockWSRepo{}
		auditRepo := &mockAuditRepo{}
		ws := &models.Workspace{Name: "WS", ManagerID: 5}
		ws.ID = 1
		wsRepo.On("FindByID", uint(1)).Return(ws, nil)
		wsRepo.On("AddMember", mock.AnythingOfType("*models.WorkspaceMember")).Return(nil)
		auditRepo.On("Append", mock.MatchedBy(func(e *models.AuditEntry) bool {
			return e.Action == models.AuditActionMemberAdded
		})).Return(nil)

		h := &handlers.WorkspaceHandler{Repo: wsRepo, UserRepo: &mockRepo{}, Audit: auditRepo}
		body := jsonBody(t, map[string]any{"user_id": 2})
		w := wsRequest(h.AddMember, http.MethodPost, "/workspaces/1/members", "/workspaces/:id/members", body, 5, "manager")

		assert.Equal(t, http.StatusCreated, w.Code)
		auditRepo.AssertExpectations(t)
	})
}
