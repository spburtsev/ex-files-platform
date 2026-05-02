package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func docsServer(tokens *mockTokens, repo *mockDocumentRepo, storage *mockStorage, issues ...*mockIssueRepo) *handlers.Server {
	var ir *mockIssueRepo
	if len(issues) > 0 && issues[0] != nil {
		ir = issues[0]
	} else {
		ir = &mockIssueRepo{}
	}
	return &handlers.Server{
		UserRepo:     &mockUserRepo{},
		Tokens:       tokens,
		Hasher:       stubHasher{},
		Audit:        &dummyAudit{},
		DocumentRepo: repo,
		Storage:      storage,
		IssueRepo:    ir,
	}
}

func multipartBody(t *testing.T, filename, contentType string, payload []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := make(map[string][]string)
	if contentType != "" {
		hdr["Content-Type"] = []string{contentType}
	}
	hdr["Content-Disposition"] = []string{
		`form-data; name="file"; filename="` + filename + `"`,
	}
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func multipartUpload(t *testing.T, url, token, filename, contentType string, payload []byte) *http.Response {
	t.Helper()
	body, ct := multipartBody(t, filename, contentType, payload)
	req, err := http.NewRequest(http.MethodPost, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return res
}

func TestDocumentsList_PaginationAndFilters(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("ListByIssue", uint(7), "report", "pending", 20, 0).Return([]models.Document{
		{Model: gormModelID(1), Name: "report-q1.pdf", IssueID: 7, Status: models.DocumentStatusPending},
	}, int64(1), nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet,
		srv.URL+"/issues/7/documents?search=report&status=pending", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "1", res.Header.Get("X-Total-Count"))
	var got oapi.ListDocumentsResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Len(t, got.Documents, 1)
	assert.Equal(t, oapi.DocumentStatusPending, got.Documents[0].Status)
}

func TestDocumentsUpload_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	storage := &mockStorage{}
	issues := &mockIssueRepo{}

	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), Resolved: false}, nil)
	repo.On("FindByIssueAndHash", uint(7), mock.AnythingOfType("string")).Return(nil, gorm.ErrRecordNotFound)
	repo.On("Create", mock.AnythingOfType("*models.Document")).Return(uint(100), nil)
	storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, int64(5), "text/plain").Return(nil)
	repo.On("CreateVersion", mock.AnythingOfType("*models.DocumentVersion")).Return(uint(200), nil)
	repo.On("FindByID", uint(100)).Return(&models.Document{
		Model:      gormModelID(100),
		Name:       "report.txt",
		MimeType:   "text/plain",
		Size:       5,
		Hash:       "abc",
		Status:     models.DocumentStatusPending,
		UploaderID: 1,
		IssueID:    7,
	}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, storage, issues))
	defer srv.Close()

	res := multipartUpload(t, srv.URL+"/issues/7/documents", "test-token", "report.txt", "text/plain", []byte("hello"))
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode, "body=%s", readBody(res))
	var got oapi.UploadDocumentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "100", got.Document.ID)
	assert.Equal(t, "report.txt", got.Document.Name)
	assert.Equal(t, oapi.DocumentStatusPending, got.Document.Status)
	assert.Equal(t, "200", got.Version.ID)
}

func TestDocumentsUpload_DuplicateHashReturns409(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), Resolved: false}, nil)
	repo.On("FindByIssueAndHash", uint(7), mock.AnythingOfType("string")).Return(&models.Document{
		Model: gormModelID(50),
		Name:  "existing.txt",
	}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues))
	defer srv.Close()

	res := multipartUpload(t, srv.URL+"/issues/7/documents", "test-token", "report.txt", "text/plain", []byte("hello"))
	defer res.Body.Close()

	assert.Equal(t, http.StatusConflict, res.StatusCode)
}

func TestDocumentsUpload_ResolvedIssueReturns422(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), Resolved: true}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues))
	defer srv.Close()

	res := multipartUpload(t, srv.URL+"/issues/7/documents", "test-token", "report.txt", "text/plain", []byte("hello"))
	defer res.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
	repo.AssertNotCalled(t, "Create", mock.Anything)
	repo.AssertNotCalled(t, "FindByIssueAndHash", mock.Anything, mock.Anything)
}

func TestDocumentsGet_IncludesVersions(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model:      gormModelID(42),
		Name:       "doc.pdf",
		MimeType:   "application/pdf",
		Status:     models.DocumentStatusPending,
		UploaderID: 1,
		IssueID:    7,
	}, nil)
	repo.On("GetVersions", uint(42)).Return([]models.DocumentVersion{
		{Model: gormModelID(101), DocumentID: 42, Version: 1, Hash: "h1", Size: 100, StorageKey: "k1"},
		{Model: gormModelID(102), DocumentID: 42, Version: 2, Hash: "h2", Size: 200, StorageKey: "k2"},
	}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetDocumentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "42", got.Document.Document.ID)
	assert.Len(t, got.Document.Versions, 2)
	assert.Equal(t, int32(2), got.Document.Versions[1].Version)
}

func TestDocumentsDelete_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{Model: gormModelID(42), Name: "x"}, nil)
	repo.On("Delete", uint(42)).Return(nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestDocumentsUploadVersion_BumpsVersionNumber(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	storage := &mockStorage{}

	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model:      gormModelID(42),
		Name:       "doc.pdf",
		MimeType:   "application/pdf",
		IssueID:    7,
		UploaderID: 1,
	}, nil)
	repo.On("LatestVersionNumber", uint(42)).Return(2, nil)
	storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, int64(5), "application/pdf").Return(nil)
	repo.On("CreateVersion", mock.AnythingOfType("*models.DocumentVersion")).Return(uint(300), nil).Run(func(a mock.Arguments) {
		v := a.Get(0).(*models.DocumentVersion)
		assert.Equal(t, 3, v.Version)
	})

	srv := newTestServer(t, docsServer(tokens, repo, storage))
	defer srv.Close()

	res := multipartUpload(t, srv.URL+"/documents/42/versions", "test-token", "doc.pdf", "application/pdf", []byte("hello"))
	defer res.Body.Close()
	assert.Equal(t, http.StatusCreated, res.StatusCode, "body=%s", readBody(res))
}

func TestDocumentsGetDownloadUrl_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	storage := &mockStorage{}

	repo.On("GetVersion", uint(101)).Return(&models.DocumentVersion{
		Model: gormModelID(101), DocumentID: 42, StorageKey: "key/abc",
	}, nil)
	storage.On("PresignedURL", mock.Anything, "key/abc", mock.Anything).Return("https://minio.example/abc", nil)

	srv := newTestServer(t, docsServer(tokens, repo, storage))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42/versions/101/download", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetDownloadUrlResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "https://minio.example/abc", got.URL.String())
}

func TestDocumentsGetFile_StreamsBytes(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	storage := &mockStorage{}

	payload := []byte("file contents")
	repo.On("GetVersion", uint(101)).Return(&models.DocumentVersion{
		Model: gormModelID(101), DocumentID: 42, StorageKey: "k", Size: int64(len(payload)),
	}, nil)
	storage.On("Get", mock.Anything, "k").Return(io.NopCloser(bytes.NewReader(payload)), nil)

	srv := newTestServer(t, docsServer(tokens, repo, storage))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42/versions/101/file", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, body)
}

func TestDocumentsSubmit_UploaderOnly(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 9, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusPending,
	}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/submit", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestDocumentsSubmit_TransitionsToInReview(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusPending,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		d := a.Get(0).(*models.Document)
		assert.Equal(t, models.DocumentStatusInReview, d.Status)
	})

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/submit", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.UpdateDocumentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, oapi.DocumentStatusInReview, got.Document.Status)
}

func TestDocumentsSubmit_InvalidTransitionReturns422(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusApproved,
	}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/submit", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
}

func TestDocumentsApprove_ManagerSucceeds(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	repo := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), Resolved: false}, nil)
	issues.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.UpdateDocumentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, oapi.DocumentStatusApproved, got.Document.Status)
}

func TestDocumentsApprove_MarksIssueResolved(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	repo := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), Resolved: false}, nil)
	issues.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil).Run(func(a mock.Arguments) {
		i := a.Get(0).(*models.Issue)
		assert.True(t, i.Resolved, "issue must be marked resolved")
	})

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	issues.AssertExpectations(t)
}

func TestDocumentsApprove_AlreadyResolvedIssueSkipsUpdate(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	repo := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), Resolved: true}, nil)
	// IssueRepo.Update should NOT be called when issue already resolved.

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	issues.AssertNotCalled(t, "Update", mock.Anything)
}

func TestDocumentsApprove_NonReviewerForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 5, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusInReview,
	}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestDocumentsReject_StoresNote(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		d := a.Get(0).(*models.Document)
		assert.Equal(t, models.DocumentStatusRejected, d.Status)
		assert.Equal(t, "fix section 3", d.ReviewerNote)
	})

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	body := strings.NewReader(`{"note":"fix section 3"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/reject", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestDocumentsRequestChanges_StoresNote(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		d := a.Get(0).(*models.Document)
		assert.Equal(t, models.DocumentStatusChangesRequested, d.Status)
		assert.Equal(t, "tighten methodology", d.ReviewerNote)
	})

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	body := strings.NewReader(`{"note":"tighten methodology"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/request-changes", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestDocumentsResubmit_FromChangesRequested(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusChangesRequested,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/resubmit", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestDocumentsAssignReviewer_ManagerOnly(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)

	srv := newTestServer(t, docsServer(tokens, &mockDocumentRepo{}, &mockStorage{}))
	defer srv.Close()

	body := strings.NewReader(`{"reviewerId":"5"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/documents/42/reviewer", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestDocumentsAssignReviewer_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 99, models.RoleManager)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		d := a.Get(0).(*models.Document)
		require.NotNil(t, d.ReviewerID)
		assert.Equal(t, uint(5), *d.ReviewerID)
	})

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}))
	defer srv.Close()

	body := strings.NewReader(`{"reviewerId":"5"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPut, srv.URL+"/documents/42/reviewer", body))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func readBody(res *http.Response) string {
	b, _ := io.ReadAll(res.Body)
	res.Body = io.NopCloser(bytes.NewReader(b))
	return string(b)
}
