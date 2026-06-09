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

// docsServer wires a document-handler test server. Pass nil for issues/ws to
// get permissive defaults: any issue resolves to a stub issue in workspace 1
// whose manager is user 1, so callers with uid 1 pass the new view checks.
func docsServer(tokens *mockTokens, repo *mockDocumentRepo, storage *mockStorage, issues *mockIssueRepo, ws *mockWorkspaceRepo) *handlers.Server {
	if issues == nil {
		issues = &mockIssueRepo{}
		issues.On("FindByID", mock.Anything).Return(&models.Issue{Model: gormModelID(1), WorkspaceID: 1, RequiredApprovals: 1}, nil).Maybe()
		issues.On("GetReviewers", mock.Anything).Return([]models.User{}, nil).Maybe()
	}
	if ws == nil {
		ws = &mockWorkspaceRepo{}
		ws.On("FindByID", mock.Anything).Return(&models.Workspace{Model: gormModelID(1), ManagerID: 1}, nil).Maybe()
		ws.On("GetMembers", mock.Anything).Return([]models.User{}, nil).Maybe()
		ws.On("IsMember", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	}
	ar := &mockDocumentApprovalRepo{}
	ar.On("Create", mock.Anything).Return(nil).Maybe()
	ar.On("ListByDocument", mock.Anything).Return([]models.DocumentApproval{}, nil).Maybe()
	ar.On("ListByDocumentIDs", mock.Anything).Return([]models.DocumentApproval{}, nil).Maybe()
	ar.On("CountByReviewers", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	ar.On("DeleteByDocument", mock.Anything).Return(nil).Maybe()
	return &handlers.Server{
		UserRepo:      &mockUserRepo{},
		Tokens:        tokens,
		Hasher:        stubHasher{},
		DocumentRepo:  repo,
		Storage:       storage,
		IssueRepo:     issues,
		WorkspaceRepo: ws,
		ApprovalRepo:  ar,
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
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), CreatorID: 1, RequiredApprovals: 1}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, nil))
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

	// caller (uid 1) is the issue creator, so canViewIssue short-circuits
	// without any workspace lookup.
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), CreatorID: 1, Resolved: false}, nil)
	repo.On("FindByIssueAndHash", uint(7), mock.AnythingOfType("string")).Return(nil, gorm.ErrRecordNotFound)
	repo.On("Create", mock.AnythingOfType("*models.Document")).Return(uint(100), nil)
	storage.On("Upload", mock.Anything, mock.AnythingOfType("string"), mock.Anything, int64(5), "text/plain").Return(nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
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

	srv := newTestServer(t, docsServer(tokens, repo, storage, issues, nil))
	defer srv.Close()

	res := multipartUpload(t, srv.URL+"/issues/7/documents", "test-token", "report.txt", "text/plain", []byte("hello"))
	defer res.Body.Close()

	require.Equal(t, http.StatusCreated, res.StatusCode, "body=%s", readBody(res))
	var got oapi.UploadDocumentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "100", got.Document.ID)
	assert.Equal(t, "report.txt", got.Document.Name)
	assert.Equal(t, oapi.DocumentStatusPending, got.Document.Status)
}

func TestDocumentsUpload_DuplicateHashReturns409(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), CreatorID: 1, Resolved: false}, nil)
	repo.On("FindByIssueAndHash", uint(7), mock.AnythingOfType("string")).Return(&models.Document{
		Model: gormModelID(50),
		Name:  "existing.txt",
	}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, nil))
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
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), CreatorID: 1, Resolved: true}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, nil))
	defer srv.Close()

	res := multipartUpload(t, srv.URL+"/issues/7/documents", "test-token", "report.txt", "text/plain", []byte("hello"))
	defer res.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
	repo.AssertNotCalled(t, "Create", mock.Anything)
	repo.AssertNotCalled(t, "FindByIssueAndHash", mock.Anything, mock.Anything)
}

func TestDocumentsGet_HappyPath(t *testing.T) {
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

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	var got oapi.GetDocumentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "42", got.Document.ID)
	assert.Equal(t, "doc.pdf", got.Document.Name)
}

// TestDocumentsGet_NonMemberNotFound: a user who is neither uploader, issue
// participant, workspace manager nor member gets 404 (existence stays hidden).
func TestDocumentsGet_NonMemberNotFound(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 5, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), Name: "doc.pdf", UploaderID: 1, IssueID: 7,
	}, nil)
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{
		Model: gormModelID(7), WorkspaceID: 3, CreatorID: 1, AssigneeID: 2,
	}, nil)
	ws := &mockWorkspaceRepo{}
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 9}, nil)
	ws.On("IsMember", uint(3), uint(5)).Return(false, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, ws))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	ws.AssertCalled(t, "IsMember", uint(3), uint(5))
}

func TestDocumentsDelete_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockDocumentRepo{}
	storage := &mockStorage{}
	// caller is not the uploader, but owns the workspace.
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), Name: "x", UploaderID: 2, IssueID: 7, StorageKey: "key/x",
	}, nil)
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3, CreatorID: 2}, nil)
	ws := &mockWorkspaceRepo{}
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 1}, nil)
	repo.On("Delete", uint(42)).Return(nil)
	storage.On("Delete", mock.Anything, "key/x").Return(nil)

	srv := newTestServer(t, docsServer(tokens, repo, storage, issues, ws))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	storage.AssertCalled(t, "Delete", mock.Anything, "key/x")
}

// TestDocumentsDelete_MemberNotUploaderForbidden: a workspace member who can
// view the document but is neither uploader nor manager gets 403 and nothing
// is deleted.
func TestDocumentsDelete_MemberNotUploaderForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 5, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	storage := &mockStorage{}
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), Name: "x", UploaderID: 1, IssueID: 7, StorageKey: "key/x",
	}, nil)
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{
		Model: gormModelID(7), WorkspaceID: 3, CreatorID: 1, AssigneeID: 2,
	}, nil)
	ws := &mockWorkspaceRepo{}
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: 9}, nil)
	ws.On("IsMember", uint(3), uint(5)).Return(true, nil)

	srv := newTestServer(t, docsServer(tokens, repo, storage, issues, ws))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	repo.AssertNotCalled(t, "Delete", mock.Anything)
	storage.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestDocumentsDelete_NotFound(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockDocumentRepo{}
	repo.On("FindByID", uint(42)).Return(nil, assert.AnError)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	repo.AssertNotCalled(t, "Delete", mock.Anything)
}

func TestDocumentsDelete_DeleteFailureReturns500(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	repo := &mockDocumentRepo{}
	// caller is the uploader, so no issue/workspace lookups happen.
	repo.On("FindByID", uint(42)).Return(&models.Document{Model: gormModelID(42), Name: "x", UploaderID: 1}, nil)
	repo.On("Delete", uint(42)).Return(assert.AnError)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodDelete, srv.URL+"/documents/42", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestDocumentsDelete_Unauthorized(t *testing.T) {
	srv := newTestServer(t, docsServer(&mockTokens{}, &mockDocumentRepo{}, &mockStorage{}, nil, nil))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/documents/42", nil)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestDocumentsGetDownloadUrl_HappyPath(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)
	repo := &mockDocumentRepo{}
	storage := &mockStorage{}

	// caller is the uploader: canViewDocument short-circuits, no lookups.
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, StorageKey: "key/abc",
	}, nil)
	storage.On("PresignedURL", mock.Anything, "key/abc", mock.Anything).Return("https://minio.example/abc", nil)

	srv := newTestServer(t, docsServer(tokens, repo, storage, nil, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42/download", nil))
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
	// caller is the uploader: canViewDocument short-circuits, no lookups.
	repo.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, StorageKey: "k", Size: int64(len(payload)),
	}, nil)
	storage.On("Get", mock.Anything, "k").Return(io.NopCloser(bytes.NewReader(payload)), nil)

	srv := newTestServer(t, docsServer(tokens, repo, storage, nil, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/documents/42/file", nil))
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

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
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

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
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

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/submit", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
}

// approveWorkspace returns a workspace mock for workspace 3 owned by the
// given manager; canReview and approvalProgressRecipients both hit it.
func approveWorkspace(managerID uint) *mockWorkspaceRepo {
	ws := &mockWorkspaceRepo{}
	ws.On("FindByID", uint(3)).Return(&models.Workspace{Model: gormModelID(3), ManagerID: managerID}, nil)
	ws.On("GetMembers", uint(3)).Return([]models.User{}, nil).Maybe()
	return ws
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
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3, Resolved: false}, nil)
	issues.On("GetReviewers", uint(7)).Return([]models.User{}, nil)
	issues.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, approveWorkspace(99)))
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
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3, Resolved: false}, nil)
	issues.On("GetReviewers", uint(7)).Return([]models.User{}, nil)
	issues.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil).Run(func(a mock.Arguments) {
		i := a.Get(0).(*models.Issue)
		assert.True(t, i.Resolved, "issue must be marked resolved")
	})

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, approveWorkspace(99)))
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
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3, Resolved: true}, nil)
	issues.On("GetReviewers", uint(7)).Return([]models.User{}, nil)
	// IssueRepo.Update should NOT be called when issue already resolved.

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, approveWorkspace(99)))
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

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
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
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		d := a.Get(0).(*models.Document)
		assert.Equal(t, models.DocumentStatusRejected, d.Status)
		assert.Equal(t, "fix section 3", d.ReviewerNote)
	})
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3}, nil)
	issues.On("GetReviewers", uint(7)).Return([]models.User{}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, approveWorkspace(99)))
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
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		d := a.Get(0).(*models.Document)
		assert.Equal(t, models.DocumentStatusChangesRequested, d.Status)
		assert.Equal(t, "tighten methodology", d.ReviewerNote)
	})
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3}, nil)
	issues.On("GetReviewers", uint(7)).Return([]models.User{}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, approveWorkspace(99)))
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

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, nil, nil))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/resubmit", nil))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestDocumentsAssignReviewer_ManagerOnly(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleEmployee)

	srv := newTestServer(t, docsServer(tokens, &mockDocumentRepo{}, &mockStorage{}, nil, nil))
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
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	repo.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		d := a.Get(0).(*models.Document)
		require.NotNil(t, d.ReviewerID)
		assert.Equal(t, uint(5), *d.ReviewerID)
	})
	issues := &mockIssueRepo{}
	issues.On("FindByID", uint(7)).Return(&models.Issue{Model: gormModelID(7), WorkspaceID: 3}, nil)

	srv := newTestServer(t, docsServer(tokens, repo, &mockStorage{}, issues, approveWorkspace(99)))
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
