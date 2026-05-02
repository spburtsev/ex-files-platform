package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockTokenService struct{ mock.Mock }

func (m *mockTokenService) Issue(user *models.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *mockTokenService) Validate(tokenStr string) (*models.Claims, error) {
	args := m.Called(tokenStr)
	if c, ok := args.Get(0).(*models.Claims); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func runMiddleware(ts *mockTokenService, req *http.Request) (status int, nextCalled bool, ctxKeys map[string]any) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	ctxKeys = make(map[string]any)
	r.GET("/test", middleware.AuthMiddleware(ts), func(c *gin.Context) {
		nextCalled = true
		ctxKeys["user_id"], _ = c.Get("user_id")
		ctxKeys["email"], _ = c.Get("email")
		ctxKeys["role"], _ = c.Get("role")
		c.Status(http.StatusOK)
	})

	r.ServeHTTP(w, req)
	return w.Code, nextCalled, ctxKeys
}

var validClaims = &models.Claims{
	UserID: 7,
	Email:  "x@y.com",
	Role:   models.RoleEmployee,
}

func TestRequestLogger(t *testing.T) {
	t.Run("logs_200", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware.RequestLogger())
		r.GET("/ok", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("logs_404", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware.RequestLogger())
		r.GET("/notfound", func(c *gin.Context) {
			c.Status(http.StatusNotFound)
		})
		req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("logs_500", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware.RequestLogger())
		r.GET("/error", func(c *gin.Context) {
			c.Status(http.StatusInternalServerError)
		})
		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("logs_with_origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(middleware.RequestLogger())
		r.GET("/with-origin", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest(http.MethodGet, "/with-origin", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		buildRequest func() *http.Request
		setup        func(ts *mockTokenService)
		wantStatus   int
		wantNext     bool
		wantUserID   any
	}{
		{
			name: "no_token_anywhere",
			buildRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test", nil)
			},
			setup:      func(ts *mockTokenService) {},
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
		},
		{
			name: "valid_cookie",
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.AddCookie(&http.Cookie{Name: "session", Value: "goodtoken"})
				return req
			},
			setup: func(ts *mockTokenService) {
				ts.On("Validate", "goodtoken").Return(validClaims, nil)
			},
			wantStatus: http.StatusOK,
			wantNext:   true,
			wantUserID: uint(7),
		},
		{
			name: "valid_bearer_header",
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Authorization", "Bearer goodtoken")
				return req
			},
			setup: func(ts *mockTokenService) {
				ts.On("Validate", "goodtoken").Return(validClaims, nil)
			},
			wantStatus: http.StatusOK,
			wantNext:   true,
			wantUserID: uint(7),
		},
		{
			name: "cookie_takes_priority_over_header",
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.AddCookie(&http.Cookie{Name: "session", Value: "cookietoken"})
				req.Header.Set("Authorization", "Bearer headertoken")
				return req
			},
			setup: func(ts *mockTokenService) {
				// must be called with cookie value, not header value
				ts.On("Validate", "cookietoken").Return(validClaims, nil)
			},
			wantStatus: http.StatusOK,
			wantNext:   true,
		},
		{
			name: "validate_returns_error",
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.AddCookie(&http.Cookie{Name: "session", Value: "badtoken"})
				return req
			},
			setup: func(ts *mockTokenService) {
				ts.On("Validate", "badtoken").Return(nil, errors.New("expired"))
			},
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
		},
		{
			name: "validate_returns_nil_claims_no_error",
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.AddCookie(&http.Cookie{Name: "session", Value: "nilclaims"})
				return req
			},
			setup: func(ts *mockTokenService) {
				ts.On("Validate", "nilclaims").Return(nil, nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantNext:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := &mockTokenService{}
			tc.setup(ts)

			status, nextCalled, keys := runMiddleware(ts, tc.buildRequest())

			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantNext, nextCalled)
			if tc.wantUserID != nil {
				assert.Equal(t, tc.wantUserID, keys["user_id"])
			}
			ts.AssertExpectations(t)
		})
	}
}
