package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

// --- Mock comment repo ---

type mockCommentRepo struct{ mock.Mock }

func (m *mockCommentRepo) Create(comment *models.Comment) error {
	args := m.Called(comment)
	comment.ID = 10
	return args.Error(0)
}

func (m *mockCommentRepo) FindByID(id uint) (*models.Comment, error) {
	args := m.Called(id)
	if c, ok := args.Get(0).(*models.Comment); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCommentRepo) ListByDocument(documentID uint) ([]models.Comment, error) {
	args := m.Called(documentID)
	if c, ok := args.Get(0).([]models.Comment); ok {
		return c, args.Error(1)
	}
	return nil, args.Error(1)
}

func commentRequest(handler gin.HandlerFunc, method, path, routePattern string, body *bytes.Buffer, userID uint) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Handle(method, routePattern, func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("role", "employee")
		handler(c)
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

func TestCommentCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockCommentRepo{}
		auditRepo := &mockAuditRepo{}
		hub := services.NewSSEHub()

		repo.On("Create", mock.AnythingOfType("*models.Comment")).Return(nil)
		repo.On("FindByID", uint(10)).Return(&models.Comment{
			Model:      gorm.Model{ID: 10, CreatedAt: time.Now()},
			DocumentID: 5,
			AuthorID:   1,
			Body:       "Great document!",
			Author:     models.User{Model: gorm.Model{ID: 1}, Name: "Alice"},
		}, nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)

		h := &handlers.CommentHandler{Repo: repo, Audit: auditRepo, Hub: hub}
		body, _ := json.Marshal(map[string]string{"body": "Great document!"})
		w := commentRequest(h.Create, http.MethodPost, "/documents/5/comments", "/documents/:id/comments", bytes.NewBuffer(body), 1)

		assert.Equal(t, http.StatusCreated, w.Code)
		repo.AssertExpectations(t)
	})

	t.Run("missing_body", func(t *testing.T) {
		repo := &mockCommentRepo{}
		hub := services.NewSSEHub()
		h := &handlers.CommentHandler{Repo: repo, Audit: &mockAuditRepo{}, Hub: hub}

		body, _ := json.Marshal(map[string]string{})
		w := commentRequest(h.Create, http.MethodPost, "/documents/5/comments", "/documents/:id/comments", bytes.NewBuffer(body), 1)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid_doc_id", func(t *testing.T) {
		repo := &mockCommentRepo{}
		hub := services.NewSSEHub()
		h := &handlers.CommentHandler{Repo: repo, Audit: &mockAuditRepo{}, Hub: hub}

		body, _ := json.Marshal(map[string]string{"body": "test"})
		w := commentRequest(h.Create, http.MethodPost, "/documents/abc/comments", "/documents/:id/comments", bytes.NewBuffer(body), 1)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCommentList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &mockCommentRepo{}
		hub := services.NewSSEHub()

		repo.On("ListByDocument", uint(5)).Return([]models.Comment{
			{
				Model:      gorm.Model{ID: 1, CreatedAt: time.Now()},
				DocumentID: 5,
				AuthorID:   1,
				Body:       "Comment 1",
				Author:     models.User{Model: gorm.Model{ID: 1}, Name: "Alice"},
			},
			{
				Model:      gorm.Model{ID: 2, CreatedAt: time.Now()},
				DocumentID: 5,
				AuthorID:   2,
				Body:       "Comment 2",
				Author:     models.User{Model: gorm.Model{ID: 2}, Name: "Bob"},
			},
		}, nil)

		h := &handlers.CommentHandler{Repo: repo, Audit: &mockAuditRepo{}, Hub: hub}
		w := commentRequest(h.List, http.MethodGet, "/documents/5/comments", "/documents/:id/comments", nil, 1)

		assert.Equal(t, http.StatusOK, w.Code)
		repo.AssertExpectations(t)
	})

	t.Run("invalid_doc_id", func(t *testing.T) {
		repo := &mockCommentRepo{}
		hub := services.NewSSEHub()
		h := &handlers.CommentHandler{Repo: repo, Audit: &mockAuditRepo{}, Hub: hub}

		w := commentRequest(h.List, http.MethodGet, "/documents/abc/comments", "/documents/:id/comments", nil, 1)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
