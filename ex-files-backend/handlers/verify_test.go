package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

func verifyRequest(handler gin.HandlerFunc, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/verify", handler)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	r.ServeHTTP(w, req)
	return w
}

func TestVerify(t *testing.T) {
	t.Run("missing_hash", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		h := &handlers.VerifyHandler{Repo: docRepo}
		w := verifyRequest(h.Verify, "/verify")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not_found", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("FindByHash", "abc123").Return(nil, gorm.ErrRecordNotFound)

		h := &handlers.VerifyHandler{Repo: docRepo}
		w := verifyRequest(h.Verify, "/verify?hash=abc123")

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, false, resp["verified"])
	})

	t.Run("found", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{
			Model:  gorm.Model{ID: 1, CreatedAt: time.Now()},
			Name:   "test.pdf",
			Hash:   "abc123",
			Status: models.DocumentStatusApproved,
		}
		docRepo.On("FindByHash", "abc123").Return(doc, nil)

		h := &handlers.VerifyHandler{Repo: docRepo}
		w := verifyRequest(h.Verify, "/verify?hash=abc123")

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, true, resp["verified"])
		assert.Equal(t, "test.pdf", resp["document_name"])
		assert.Equal(t, string(models.DocumentStatusApproved), resp["status"])
	})
}
