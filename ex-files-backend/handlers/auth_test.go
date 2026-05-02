package handlers_test

import (
	"bytes"
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

func TestAuthRegister_HappyPath(t *testing.T) {
	users := &mockUserRepo{}
	tokens := &mockTokens{}

	users.On("FindByEmail", "alice@example.com").Return(nil, errors.New("not found"))
	users.On("Create", mock.AnythingOfType("*models.User")).Return(uint(42), nil).Run(func(a mock.Arguments) {
		u := a.Get(0).(*models.User)
		u.ID = 42
		u.CreatedAt = time.Now()
	})
	tokens.On("Issue", mock.AnythingOfType("*models.User")).Return("test-token", nil)

	s := &handlers.Server{
		UserRepo: users,
		Tokens:   tokens,
		Hasher:   stubHasher{},
		Audit:    &dummyAudit{},
	}
	srv := newTestServer(t, s)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "secret-p455word",
		"name":     "Alice",
	})
	res, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusCreated, res.StatusCode)

	var got oapi.AuthResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "alice@example.com", got.User.Email)
	assert.Equal(t, "Alice", got.User.Name)
	assert.Equal(t, "42", got.User.ID)
	assert.Equal(t, "test-token", got.Token)

	cookie := findCookie(res.Cookies(), "session")
	require.NotNil(t, cookie)
	assert.Equal(t, "test-token", cookie.Value)
	assert.True(t, cookie.HttpOnly)
}

func TestAuthRegister_DuplicateEmailReturns409(t *testing.T) {
	users := &mockUserRepo{}
	users.On("FindByEmail", "alice@example.com").Return(&models.User{Email: "alice@example.com"}, nil)

	s := &handlers.Server{UserRepo: users, Tokens: &mockTokens{}, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{
		"email":    "alice@example.com",
		"password": "secret-p455word",
		"name":     "Alice",
	})
	res, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusConflict, res.StatusCode)

	var got oapi.Error
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Contains(t, got.Error, "already registered")
}

func TestAuthLogin_WrongPasswordReturns401(t *testing.T) {
	users := &mockUserRepo{}
	users.On("FindByEmail", "alice@example.com").Return(&models.User{
		Email:        "alice@example.com",
		PasswordHash: "hashed:correct-password",
	}, nil)

	s := &handlers.Server{UserRepo: users, Tokens: &mockTokens{}, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	body := strings.NewReader(`{"email":"alice@example.com","password":"wrong-password"}`)
	res, err := http.Post(srv.URL+"/auth/login", "application/json", body)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuthMe_RoundtripsBearerAuth(t *testing.T) {
	users := &mockUserRepo{}
	tokens := &mockTokens{}

	tokens.On("Validate", "good-token").Return(&models.Claims{
		UserID: 7,
		Email:  "alice@example.com",
		Role:   models.RoleEmployee,
	}, nil)
	users.On("FindByID", uint(7)).Return(&models.User{
		Email: "alice@example.com",
		Name:  "Alice",
		Role:  models.RoleEmployee,
	}, nil)

	s := &handlers.Server{UserRepo: users, Tokens: tokens, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)

	var got oapi.MeResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "alice@example.com", got.User.Email)
	assert.Equal(t, oapi.RoleEmployee, got.User.Role)
}

func TestAuthMe_NoTokenReturns401(t *testing.T) {
	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: &mockTokens{}, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/auth/me")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuthLogout_ClearsCookie(t *testing.T) {
	tokens := &mockTokens{}
	tokens.On("Validate", "good-token").Return(&models.Claims{UserID: 1, Role: models.RoleEmployee}, nil)
	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: tokens, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "good-token"})
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	cookie := findCookie(res.Cookies(), "session")
	require.NotNil(t, cookie)
	assert.True(t, cookie.MaxAge < 0, "cookie should be invalidated")
}

func TestAuthListUsers_RequiresAuth(t *testing.T) {
	s := &handlers.Server{UserRepo: &mockUserRepo{}, Tokens: &mockTokens{}, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/auth/users")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuthListUsers_ReturnsAll(t *testing.T) {
	users := &mockUserRepo{}
	tokens := &mockTokens{}

	tokens.On("Validate", "test-token").Return(&models.Claims{UserID: 1, Role: models.RoleManager}, nil)
	users.On("ListAll").Return([]models.User{
		{Email: "a@x", Name: "A", Role: models.RoleEmployee},
		{Email: "b@x", Name: "B", Role: models.RoleManager},
	}, nil)

	s := &handlers.Server{UserRepo: users, Tokens: tokens, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/auth/users", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetUsersResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Users, 2)
}

func TestAuthForgotPassword_AlwaysReturns200(t *testing.T) {
	users := &mockUserRepo{}
	users.On("FindByEmail", "nobody@example.com").Return(nil, errors.New("not found"))

	s := &handlers.Server{UserRepo: users, Tokens: &mockTokens{}, Hasher: stubHasher{}, Audit: &dummyAudit{}}
	srv := newTestServer(t, s)
	defer srv.Close()

	body := strings.NewReader(`{"email":"nobody@example.com"}`)
	res, err := http.Post(srv.URL+"/auth/forgot-password", "application/json", body)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	var msg oapi.MessageResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&msg))
	assert.Contains(t, msg.Message, "if the email exists")
}
