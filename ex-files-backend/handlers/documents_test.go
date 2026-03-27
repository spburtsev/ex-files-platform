package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	docsv1 "github.com/spburtsev/ex-files-backend/gen/documents/v1"
	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

// --- Mock document repo ---

type mockDocRepo struct{ mock.Mock }

func (m *mockDocRepo) Create(doc *models.Document) error {
	args := m.Called(doc)
	doc.ID = 1
	return args.Error(0)
}

func (m *mockDocRepo) FindByID(id uint) (*models.Document, error) {
	args := m.Called(id)
	if d, ok := args.Get(0).(*models.Document); ok {
		return d, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDocRepo) Update(doc *models.Document) error {
	return m.Called(doc).Error(0)
}

func (m *mockDocRepo) ListByIssue(issueID uint, search, status string, limit, offset int) ([]models.Document, int64, error) {
	args := m.Called(issueID, search, status, limit, offset)
	return args.Get(0).([]models.Document), args.Get(1).(int64), args.Error(2)
}

func (m *mockDocRepo) Delete(id uint) error {
	return m.Called(id).Error(0)
}

func (m *mockDocRepo) CreateVersion(v *models.DocumentVersion) error {
	args := m.Called(v)
	v.ID = 1
	return args.Error(0)
}

func (m *mockDocRepo) GetVersions(documentID uint) ([]models.DocumentVersion, error) {
	args := m.Called(documentID)
	return args.Get(0).([]models.DocumentVersion), args.Error(1)
}

func (m *mockDocRepo) GetVersion(id uint) (*models.DocumentVersion, error) {
	args := m.Called(id)
	if v, ok := args.Get(0).(*models.DocumentVersion); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockDocRepo) LatestVersionNumber(documentID uint) (int, error) {
	args := m.Called(documentID)
	return args.Int(0), args.Error(1)
}

// --- Mock storage ---

type mockStorage struct{ mock.Mock }

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return m.Called(ctx, key, reader, size, contentType).Error(0)
}

func (m *mockStorage) PresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	args := m.Called(ctx, key, expires)
	return args.String(0), args.Error(1)
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

// --- helpers ---

func newDocHandler(docRepo *mockDocRepo, storage *mockStorage, auditRepo *mockAuditRepo) *handlers.DocumentHandler {
	return &handlers.DocumentHandler{Repo: docRepo, Storage: storage, Audit: auditRepo}
}

func docRequest(h gin.HandlerFunc, method, path, routePattern string, body *bytes.Buffer, contentType string, userID uint, role string) *httptest.ResponseRecorder {
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
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	r.ServeHTTP(w, req)
	return w
}

func createMultipartFile(t *testing.T, fieldName, fileName, content string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = io.WriteString(part, content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

// --- TestDocumentList ---

func TestDocumentList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docs := []models.Document{
			{Name: "doc1.pdf", Uploader: models.User{Name: "Alice"}},
			{Name: "doc2.pdf", Uploader: models.User{Name: "Bob"}},
		}
		docRepo.On("ListByIssue", uint(1), "", "", 20, 0).Return(docs, int64(2), nil)

		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.List, http.MethodGet, "/issues/1/documents", "/issues/:id/documents", nil, "", 1, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "2", w.Header().Get("X-Total-Count"))

		var resp docsv1.ListDocumentsResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp.Documents, 2)
		docRepo.AssertExpectations(t)
	})

	t.Run("with_search_and_status", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("ListByIssue", uint(1), "report", "approved", 20, 0).Return(
			[]models.Document{}, int64(0), nil,
		)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.List, http.MethodGet, "/issues/1/documents?search=report&status=approved", "/issues/:id/documents", nil, "", 1, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		docRepo.AssertExpectations(t)
	})

	t.Run("db_failure", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("ListByIssue", uint(1), "", "", 20, 0).Return(
			[]models.Document(nil), int64(0), errors.New("db error"),
		)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.List, http.MethodGet, "/issues/1/documents", "/issues/:id/documents", nil, "", 1, "manager")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- TestDocumentGet ---

func TestDocumentGet(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Get, http.MethodGet, "/documents/99", "/documents/:id", nil, "", 1, "manager")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{Name: "test.pdf", Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("GetVersions", uint(1)).Return([]models.DocumentVersion{
			{Version: 1, Hash: "abc", Uploader: models.User{Name: "Alice"}},
		}, nil)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Get, http.MethodGet, "/documents/1", "/documents/:id", nil, "", 1, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		var resp docsv1.GetDocumentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		require.NotNil(t, resp.Document)
		assert.NotNil(t, resp.Document.Document)
		assert.Len(t, resp.Document.Versions, 1)
		docRepo.AssertExpectations(t)
	})
}

// --- TestDocumentUpload ---

func TestDocumentUpload(t *testing.T) {
	t.Run("no_file", func(t *testing.T) {
		h := newDocHandler(&mockDocRepo{}, &mockStorage{}, nil)
		w := docRequest(h.Upload, http.MethodPost, "/issues/1/documents", "/issues/:id/documents", nil, "", 1, "manager")

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		storage := &mockStorage{}
		auditRepo := &mockAuditRepo{}

		docRepo.On("Create", mock.AnythingOfType("*models.Document")).Return(nil)
		storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Return(nil)
		docRepo.On("CreateVersion", mock.AnythingOfType("*models.DocumentVersion")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		docRepo.On("FindByID", uint(1)).Return(&models.Document{
			Name:     "test.pdf",
			Uploader: models.User{Name: "Alice"},
		}, nil)

		h := newDocHandler(docRepo, storage, auditRepo)

		body, contentType := createMultipartFile(t, "file", "test.pdf", "fake pdf content")
		w := docRequest(h.Upload, http.MethodPost, "/issues/1/documents", "/issues/:id/documents", body, contentType, 1, "manager")

		assert.Equal(t, http.StatusCreated, w.Code)
		docRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
	})
}

// --- TestDocumentDownload ---

func TestDocumentDownload(t *testing.T) {
	t.Run("version_not_found", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("GetVersion", uint(99)).Return(nil, gorm.ErrRecordNotFound)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Download, http.MethodGet, "/documents/1/versions/99/download", "/documents/:id/versions/:versionId/download", nil, "", 1, "manager")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		storage := &mockStorage{}
		version := &models.DocumentVersion{StorageKey: "workspaces/1/documents/1/v1/test.pdf"}
		version.ID = 1
		docRepo.On("GetVersion", uint(1)).Return(version, nil)
		storage.On("PresignedURL", mock.Anything, "workspaces/1/documents/1/v1/test.pdf", 15*time.Minute).Return("https://minio.local/signed-url", nil)

		h := newDocHandler(docRepo, storage, nil)
		w := docRequest(h.Download, http.MethodGet, "/documents/1/versions/1/download", "/documents/:id/versions/:versionId/download", nil, "", 1, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		var resp docsv1.GetDownloadURLResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp.Url, "signed-url")
		docRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
	})
}

// --- TestDocumentDelete ---

func TestDocumentDelete(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Delete, http.MethodDelete, "/documents/99", "/documents/:id", nil, "", 1, "manager")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Name: "test.pdf", Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Delete", uint(1)).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)

		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)
		w := docRequest(h.Delete, http.MethodDelete, "/documents/1", "/documents/:id", nil, "", 1, "manager")

		assert.Equal(t, http.StatusOK, w.Code)
		docRepo.AssertExpectations(t)
	})
}

// --- TestDocumentSubmit ---

func TestDocumentSubmit(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("FindByID", uint(99)).Return(nil, errors.New("not found"))
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Submit, http.MethodPost, "/documents/99/submit", "/documents/:id/submit", nil, "", 1, "employee")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("forbidden_not_uploader", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{Status: models.DocumentStatusPending, UploaderID: 2}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		// user 1 tries to submit doc uploaded by user 2
		w := docRequest(h.Submit, http.MethodPost, "/documents/1/submit", "/documents/:id/submit", nil, "", 1, "employee")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid_transition", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{Status: models.DocumentStatusApproved, UploaderID: 1}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Submit, http.MethodPost, "/documents/1/submit", "/documents/:id/submit", nil, "", 1, "employee")
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Status: models.DocumentStatusPending, UploaderID: 1, Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)
		w := docRequest(h.Submit, http.MethodPost, "/documents/1/submit", "/documents/:id/submit", nil, "", 1, "employee")
		assert.Equal(t, http.StatusOK, w.Code)
		docRepo.AssertExpectations(t)
	})
}

// --- TestDocumentApprove ---

func TestDocumentApprove(t *testing.T) {
	reviewerID := uint(5)

	t.Run("forbidden_not_reviewer", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{Status: models.DocumentStatusInReview, UploaderID: 2, ReviewerID: &reviewerID}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		// user 1 is not reviewer (5) and not manager
		w := docRequest(h.Approve, http.MethodPost, "/documents/1/approve", "/documents/:id/approve", nil, "", 1, "employee")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid_transition_from_pending", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{Status: models.DocumentStatusPending, UploaderID: 2}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Approve, http.MethodPost, "/documents/1/approve", "/documents/:id/approve", nil, "", 1, "manager")
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("success_as_manager", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Status: models.DocumentStatusInReview, UploaderID: 2, Uploader: models.User{Name: "Bob"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)
		w := docRequest(h.Approve, http.MethodPost, "/documents/1/approve", "/documents/:id/approve", nil, "", 1, "manager")
		assert.Equal(t, http.StatusOK, w.Code)

		var resp docsv1.UpdateDocumentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		require.NotNil(t, resp.Document)
		assert.Equal(t, "approved", resp.Document.Status)
		docRepo.AssertExpectations(t)
	})

	t.Run("success_as_assigned_reviewer", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Status: models.DocumentStatusInReview, UploaderID: 2, ReviewerID: &reviewerID, Uploader: models.User{Name: "Bob"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)
		// user 5 is the assigned reviewer
		w := docRequest(h.Approve, http.MethodPost, "/documents/1/approve", "/documents/:id/approve", nil, "", reviewerID, "employee")
		assert.Equal(t, http.StatusOK, w.Code)
		docRepo.AssertExpectations(t)
	})
}

// --- TestDocumentReject ---

func TestDocumentReject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Status: models.DocumentStatusInReview, UploaderID: 2, Uploader: models.User{Name: "Bob"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)

		body := bytes.NewBufferString(`{"note":"Does not meet requirements"}`)
		w := docRequest(h.Reject, http.MethodPost, "/documents/1/reject", "/documents/:id/reject", body, "application/json", 1, "manager")
		assert.Equal(t, http.StatusOK, w.Code)

		var resp docsv1.UpdateDocumentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		require.NotNil(t, resp.Document)
		assert.Equal(t, "rejected", resp.Document.Status)
		docRepo.AssertExpectations(t)
	})
}

// --- TestDocumentRequestChanges ---

func TestDocumentRequestChanges(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Status: models.DocumentStatusInReview, UploaderID: 2, Uploader: models.User{Name: "Bob"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)

		body := bytes.NewBufferString(`{"note":"Please fix section 2"}`)
		w := docRequest(h.RequestChanges, http.MethodPost, "/documents/1/request-changes", "/documents/:id/request-changes", body, "application/json", 1, "manager")
		assert.Equal(t, http.StatusOK, w.Code)

		var resp docsv1.UpdateDocumentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		require.NotNil(t, resp.Document)
		assert.Equal(t, "changes_requested", resp.Document.Status)
		docRepo.AssertExpectations(t)
	})
}

// --- TestDocumentResubmit ---

func TestDocumentResubmit(t *testing.T) {
	t.Run("forbidden_not_uploader", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{Status: models.DocumentStatusChangesRequested, UploaderID: 2}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.Resubmit, http.MethodPost, "/documents/1/resubmit", "/documents/:id/resubmit", nil, "", 1, "employee")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Status: models.DocumentStatusChangesRequested, UploaderID: 1, Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)
		w := docRequest(h.Resubmit, http.MethodPost, "/documents/1/resubmit", "/documents/:id/resubmit", nil, "", 1, "employee")
		assert.Equal(t, http.StatusOK, w.Code)

		var resp docsv1.UpdateDocumentResponse
		require.NoError(t, proto.Unmarshal(w.Body.Bytes(), &resp))
		require.NotNil(t, resp.Document)
		assert.Equal(t, "in_review", resp.Document.Status)
		docRepo.AssertExpectations(t)
	})
}

// --- TestAssignReviewer ---

func TestAssignReviewer(t *testing.T) {
	t.Run("forbidden_employee", func(t *testing.T) {
		h := newDocHandler(&mockDocRepo{}, &mockStorage{}, nil)
		body := bytes.NewBufferString(`{"reviewer_id":5}`)
		w := docRequest(h.AssignReviewer, http.MethodPut, "/documents/1/reviewer", "/documents/:id/reviewer", body, "application/json", 1, "employee")
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		auditRepo := &mockAuditRepo{}
		doc := &models.Document{Status: models.DocumentStatusInReview, UploaderID: 2, Uploader: models.User{Name: "Bob"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)
		h := newDocHandler(docRepo, &mockStorage{}, auditRepo)

		body := bytes.NewBufferString(`{"reviewer_id":5}`)
		w := docRequest(h.AssignReviewer, http.MethodPut, "/documents/1/reviewer", "/documents/:id/reviewer", body, "application/json", 1, "manager")
		assert.Equal(t, http.StatusOK, w.Code)
		docRepo.AssertExpectations(t)
	})
}

// --- TestDocumentUploadVersion ---

func TestDocumentUploadVersion(t *testing.T) {
	t.Run("document_not_found", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		docRepo.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)
		h := newDocHandler(docRepo, &mockStorage{}, nil)
		w := docRequest(h.UploadVersion, http.MethodPost, "/documents/99/versions", "/documents/:id/versions", nil, "", 1, "manager")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		storage := &mockStorage{}
		auditRepo := &mockAuditRepo{}

		doc := &models.Document{Name: "test.pdf", IssueID: 1, Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("LatestVersionNumber", uint(1)).Return(1, nil)
		storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Return(nil)
		docRepo.On("CreateVersion", mock.AnythingOfType("*models.DocumentVersion")).Return(nil)
		auditRepo.On("Append", mock.AnythingOfType("*models.AuditEntry")).Return(nil)

		h := newDocHandler(docRepo, storage, auditRepo)
		body, contentType := createMultipartFile(t, "file", "test_v2.pdf", "updated content")
		w := docRequest(h.UploadVersion, http.MethodPost, "/documents/1/versions", "/documents/:id/versions", body, contentType, 1, "manager")

		assert.Equal(t, http.StatusCreated, w.Code)
		docRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
		auditRepo.AssertExpectations(t)
	})

	t.Run("missing_file", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		doc := &models.Document{Name: "test.pdf", IssueID: 1, Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)

		h := newDocHandler(docRepo, &mockStorage{}, nil)
		// Send request with no multipart body
		w := docRequest(h.UploadVersion, http.MethodPost, "/documents/1/versions", "/documents/:id/versions", nil, "", 1, "manager")

		assert.Equal(t, http.StatusBadRequest, w.Code)
		docRepo.AssertExpectations(t)
	})

	t.Run("storage_upload_failure", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		storage := &mockStorage{}

		doc := &models.Document{Name: "test.pdf", IssueID: 1, Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("LatestVersionNumber", uint(1)).Return(1, nil)
		storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Return(errors.New("storage error"))

		h := newDocHandler(docRepo, storage, nil)
		body, contentType := createMultipartFile(t, "file", "test_v2.pdf", "updated content")
		w := docRequest(h.UploadVersion, http.MethodPost, "/documents/1/versions", "/documents/:id/versions", body, contentType, 1, "manager")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		docRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
	})

	t.Run("create_version_failure", func(t *testing.T) {
		docRepo := &mockDocRepo{}
		storage := &mockStorage{}

		doc := &models.Document{Name: "test.pdf", IssueID: 1, Uploader: models.User{Name: "Alice"}}
		doc.ID = 1
		docRepo.On("FindByID", uint(1)).Return(doc, nil)
		docRepo.On("LatestVersionNumber", uint(1)).Return(1, nil)
		storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string")).Return(nil)
		docRepo.On("CreateVersion", mock.AnythingOfType("*models.DocumentVersion")).Return(errors.New("db error"))

		h := newDocHandler(docRepo, storage, nil)
		body, contentType := createMultipartFile(t, "file", "test_v2.pdf", "updated content")
		w := docRequest(h.UploadVersion, http.MethodPost, "/documents/1/versions", "/documents/:id/versions", body, contentType, 1, "manager")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		docRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
	})
}
