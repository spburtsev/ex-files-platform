package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
)

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

var validClaims = &models.Claims{
	UserID: 7,
	Email:  "x@y.com",
	Role:   models.RoleEmployee,
}

func runAuth(ts *mockTokenService, req *http.Request) (status int, nextCalled bool, gotUserID any) {
	w := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
			gotUserID = uid
		}
		w.WriteHeader(http.StatusOK)
	})
	middleware.RequireAuth(ts)(next).ServeHTTP(w, req)
	return w.Code, nextCalled, gotUserID
}

func TestRequireAuth(t *testing.T) {
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

			status, nextCalled, uid := runAuth(ts, tc.buildRequest())

			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantNext, nextCalled)
			if tc.wantUserID != nil {
				assert.Equal(t, tc.wantUserID, uid)
			}
			ts.AssertExpectations(t)
		})
	}
}

func TestRequestLogger(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		setOrigin bool
	}{
		{"logs_200", http.StatusOK, false},
		{"logs_404", http.StatusNotFound, false},
		{"logs_500", http.StatusInternalServerError, false},
		{"logs_with_origin", http.StatusOK, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h := middleware.RequestLogger()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.setOrigin {
				req.Header.Set("Origin", "http://localhost:5173")
			}
			h.ServeHTTP(w, req)
			assert.Equal(t, tc.status, w.Code)
		})
	}
}
