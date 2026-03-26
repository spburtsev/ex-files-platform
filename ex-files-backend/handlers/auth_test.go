package handlers_test

import (
	"bytes"
	"encoding/json"
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

	authv1 "github.com/spburtsev/ex-files-backend/gen/auth/v1"
	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- Mocks ---

type mockRepo struct{ mock.Mock }

func (m *mockRepo) FindByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if u, ok := args.Get(0).(*models.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepo) FindByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if u, ok := args.Get(0).(*models.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockRepo) Create(user *models.User) error {
	args := m.Called(user)
	user.ID = 1 // simulate DB-assigned ID
	return args.Error(0)
}

func (m *mockRepo) ListAll() ([]models.User, error) {
	args := m.Called()
	if u, ok := args.Get(0).([]models.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

type mockTokens struct{ mock.Mock }

func (m *mockTokens) Issue(user *models.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *mockTokens) Validate(tokenStr string) (*models.Claims, error) {
	args := m.Called(tokenStr)
	if c, ok := args.Get(0).(*models.Claims); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

type mockHasher struct{ mock.Mock }

func (m *mockHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *mockHasher) Compare(hash, password string) error {
	return m.Called(hash, password).Error(0)
}

// --- Helpers ---

func newHandler(repo *mockRepo, tokens *mockTokens, hasher *mockHasher) *handlers.AuthHandler {
	return &handlers.AuthHandler{Repo: repo, Tokens: tokens, Hasher: hasher}
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

func executeRequest(handler gin.HandlerFunc, method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.Handle(method, path, handler)
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, http.NoBody)
	}
	c.Request = req
	r.ServeHTTP(w, req)
	return w
}

// --- TestRegister ---

func TestRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setup      func(repo *mockRepo, tokens *mockTokens, hasher *mockHasher)
		wantStatus int
		wantCookie bool
	}{
		{
			name:       "invalid_json",
			body:       "not-json",
			setup:      func(r *mockRepo, ts *mockTokens, h *mockHasher) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing_name",
			body:       map[string]string{"email": "a@b.com", "password": "password1"},
			setup:      func(r *mockRepo, ts *mockTokens, h *mockHasher) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "email_already_exists",
			body: map[string]string{"email": "a@b.com", "password": "password1", "name": "Alice"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				r.On("FindByEmail", "a@b.com").Return(&models.User{Email: "a@b.com"}, nil)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "hasher_failure",
			body: map[string]string{"email": "a@b.com", "password": "password1", "name": "Alice"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				r.On("FindByEmail", "a@b.com").Return(nil, gorm.ErrRecordNotFound)
				h.On("Hash", "password1").Return("", errors.New("hash error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "db_create_failure",
			body: map[string]string{"email": "a@b.com", "password": "password1", "name": "Alice"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				r.On("FindByEmail", "a@b.com").Return(nil, gorm.ErrRecordNotFound)
				h.On("Hash", "password1").Return("hashed", nil)
				r.On("Create", mock.AnythingOfType("*models.User")).Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "token_issue_failure",
			body: map[string]string{"email": "a@b.com", "password": "password1", "name": "Alice"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				r.On("FindByEmail", "a@b.com").Return(nil, gorm.ErrRecordNotFound)
				h.On("Hash", "password1").Return("hashed", nil)
				r.On("Create", mock.AnythingOfType("*models.User")).Return(nil)
				ts.On("Issue", mock.AnythingOfType("*models.User")).Return("", errors.New("sign error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "success",
			body: map[string]string{"email": "a@b.com", "password": "password1", "name": "Alice"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				r.On("FindByEmail", "a@b.com").Return(nil, gorm.ErrRecordNotFound)
				h.On("Hash", "password1").Return("hashed", nil)
				r.On("Create", mock.AnythingOfType("*models.User")).Return(nil)
				ts.On("Issue", mock.AnythingOfType("*models.User")).Return("tok123", nil)
			},
			wantStatus: http.StatusCreated,
			wantCookie: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			tokens := &mockTokens{}
			hasher := &mockHasher{}
			tc.setup(repo, tokens, hasher)

			h := newHandler(repo, tokens, hasher)
			var body *bytes.Buffer
			if s, ok := tc.body.(string); ok {
				body = bytes.NewBufferString(s)
			} else {
				body = jsonBody(t, tc.body)
			}

			w := executeRequest(h.Register, http.MethodPost, "/auth/register", body)

			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantCookie {
				assert.Contains(t, w.Header().Get("Set-Cookie"), "session=tok123")
			}
			repo.AssertExpectations(t)
			tokens.AssertExpectations(t)
			hasher.AssertExpectations(t)
		})
	}
}

// --- TestLogin ---

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		setup      func(repo *mockRepo, tokens *mockTokens, hasher *mockHasher)
		wantStatus int
		wantCookie bool
	}{
		{
			name:       "invalid_json",
			body:       "not-json",
			setup:      func(r *mockRepo, ts *mockTokens, h *mockHasher) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "user_not_found",
			body: map[string]string{"email": "a@b.com", "password": "password1"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				r.On("FindByEmail", "a@b.com").Return(nil, gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "wrong_password",
			body: map[string]string{"email": "a@b.com", "password": "wrong"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				r.On("FindByEmail", "a@b.com").Return(&models.User{Email: "a@b.com", PasswordHash: "hash"}, nil)
				h.On("Compare", "hash", "wrong").Return(errors.New("mismatch"))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "token_issue_failure",
			body: map[string]string{"email": "a@b.com", "password": "password1"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				u := &models.User{Email: "a@b.com", PasswordHash: "hash"}
				r.On("FindByEmail", "a@b.com").Return(u, nil)
				h.On("Compare", "hash", "password1").Return(nil)
				ts.On("Issue", u).Return("", errors.New("sign error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "success",
			body: map[string]string{"email": "a@b.com", "password": "password1"},
			setup: func(r *mockRepo, ts *mockTokens, h *mockHasher) {
				u := &models.User{Email: "a@b.com", PasswordHash: "hash"}
				r.On("FindByEmail", "a@b.com").Return(u, nil)
				h.On("Compare", "hash", "password1").Return(nil)
				ts.On("Issue", u).Return("tok456", nil)
			},
			wantStatus: http.StatusOK,
			wantCookie: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			tokens := &mockTokens{}
			hasher := &mockHasher{}
			tc.setup(repo, tokens, hasher)

			h := newHandler(repo, tokens, hasher)
			var body *bytes.Buffer
			if s, ok := tc.body.(string); ok {
				body = bytes.NewBufferString(s)
			} else {
				body = jsonBody(t, tc.body)
			}

			w := executeRequest(h.Login, http.MethodPost, "/auth/login", body)

			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantCookie {
				assert.Contains(t, w.Header().Get("Set-Cookie"), "session=tok456")
			}
			repo.AssertExpectations(t)
			tokens.AssertExpectations(t)
			hasher.AssertExpectations(t)
		})
	}
}

// --- TestMe ---

func TestMe(t *testing.T) {
	tests := []struct {
		name       string
		userID     uint
		setup      func(repo *mockRepo)
		wantStatus int
		wantEmail  string
	}{
		{
			name:   "user_not_found",
			userID: 99,
			setup: func(r *mockRepo) {
				r.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "success_employee",
			userID: 1,
			setup: func(r *mockRepo) {
				r.On("FindByID", uint(1)).Return(&models.User{Email: "a@b.com", Role: models.RoleEmployee}, nil)
			},
			wantStatus: http.StatusOK,
			wantEmail:  "a@b.com",
		},
		{
			name:   "success_manager",
			userID: 2,
			setup: func(r *mockRepo) {
				r.On("FindByID", uint(2)).Return(&models.User{Email: "m@b.com", Role: models.RoleManager}, nil)
			},
			wantStatus: http.StatusOK,
			wantEmail:  "m@b.com",
		},
		{
			name:   "success_root",
			userID: 3,
			setup: func(r *mockRepo) {
				r.On("FindByID", uint(3)).Return(&models.User{Email: "r@b.com", Role: models.RoleRoot}, nil)
			},
			wantStatus: http.StatusOK,
			wantEmail:  "r@b.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepo{}
			tc.setup(repo)

			h := newHandler(repo, &mockTokens{}, &mockHasher{})

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)
			router.GET("/auth/me", func(ctx *gin.Context) {
				ctx.Set("user_id", tc.userID)
				h.Me(ctx)
			})
			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
			c.Request = req
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantEmail != "" {
				var resp authv1.MeResponse
				require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, tc.wantEmail, resp.User.Email)
			}
			repo.AssertExpectations(t)
		})
	}
}

// --- TestLogout ---

func TestLogout(t *testing.T) {
	h := newHandler(&mockRepo{}, &mockTokens{}, &mockHasher{})
	w := executeRequest(h.Logout, http.MethodPost, "/auth/logout", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	// Cookie should be cleared (max-age=0 or negative)
	cookie := w.Header().Get("Set-Cookie")
	assert.Contains(t, cookie, "session=")
	assert.Contains(t, cookie, "Max-Age=0")
}
