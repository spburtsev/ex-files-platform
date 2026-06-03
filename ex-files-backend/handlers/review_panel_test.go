package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/models"
)

func reviewServer(tokens *mockTokens, docs *mockDocumentRepo, issues *mockIssueRepo, approvals *mockDocumentApprovalRepo) *handlers.Server {
	return &handlers.Server{
		UserRepo:     &mockUserRepo{},
		Tokens:       tokens,
		Hasher:       stubHasher{},
		DocumentRepo: docs,
		IssueRepo:    issues,
		ApprovalRepo: approvals,
		Storage:      &mockStorage{},
	}
}

func usersWithIDs(ids ...uint) []models.User {
	out := make([]models.User, 0, len(ids))
	for _, id := range ids {
		out = append(out, models.User{Model: gormModelID(id)})
	}
	return out
}

func panelIssue(creatorID uint, requiredApprovals int, resolved bool) *models.Issue {
	return &models.Issue{
		Model:             gormModelID(7),
		CreatorID:         creatorID,
		WorkspaceID:       3,
		RequiredApprovals: requiredApprovals,
		Resolved:          resolved,
	}
}

func TestDocumentsApprove_PanelPartialDoesNotResolve(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 5, models.RoleEmployee) // a panel reviewer
	docs := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	approvals := &mockDocumentApprovalRepo{}

	docs.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	issues.On("FindByID", uint(7)).Return(panelIssue(99, 2, false), nil)
	issues.On("GetReviewers", uint(7)).Return(usersWithIDs(5, 6, 7), nil)
	approvals.On("Create", mock.Anything).Return(nil)
	approvals.On("CountByReviewers", uint(42), mock.Anything).Return(int64(1), nil)
	approvals.On("ListByDocument", uint(42)).Return([]models.DocumentApproval{
		{Model: gormModelID(1), DocumentID: 42, ReviewerID: 5},
	}, nil)

	srv := newTestServer(t, reviewServer(tokens, docs, issues, approvals))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	approvals.AssertCalled(t, "Create", mock.Anything)
	issues.AssertNotCalled(t, "Update", mock.Anything) // not resolved yet
	docs.AssertNotCalled(t, "Update", mock.Anything)   // stays in_review, no status change
}

func TestDocumentsApprove_PanelReachesThresholdResolves(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 6, models.RoleEmployee) // the second reviewer
	docs := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	approvals := &mockDocumentApprovalRepo{}

	docs.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	docs.On("Update", mock.AnythingOfType("*models.Document")).Return(nil)
	issues.On("FindByID", uint(7)).Return(panelIssue(99, 2, false), nil)
	issues.On("GetReviewers", uint(7)).Return(usersWithIDs(5, 6, 7), nil)
	resolved := false
	issues.On("Update", mock.AnythingOfType("*models.Issue")).Return(nil).Run(func(a mock.Arguments) {
		resolved = a.Get(0).(*models.Issue).Resolved
	})
	approvals.On("Create", mock.Anything).Return(nil)
	approvals.On("CountByReviewers", uint(42), mock.Anything).Return(int64(2), nil)
	approvals.On("ListByDocument", uint(42)).Return([]models.DocumentApproval{
		{Model: gormModelID(1), DocumentID: 42, ReviewerID: 5},
		{Model: gormModelID(2), DocumentID: 42, ReviewerID: 6},
	}, nil)

	srv := newTestServer(t, reviewServer(tokens, docs, issues, approvals))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	issues.AssertCalled(t, "Update", mock.Anything)
	assert.True(t, resolved, "issue must be resolved once threshold reached")
}

func TestDocumentsApprove_DuplicateApprovalStaysPartial(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 5, models.RoleEmployee)
	docs := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	approvals := &mockDocumentApprovalRepo{}

	docs.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	issues.On("FindByID", uint(7)).Return(panelIssue(99, 2, false), nil)
	issues.On("GetReviewers", uint(7)).Return(usersWithIDs(5, 6, 7), nil)
	approvals.On("Create", mock.Anything).Return(nil) // idempotent no-op at repo layer
	approvals.On("CountByReviewers", uint(42), mock.Anything).Return(int64(1), nil)
	approvals.On("ListByDocument", uint(42)).Return([]models.DocumentApproval{
		{Model: gormModelID(1), DocumentID: 42, ReviewerID: 5},
	}, nil)

	srv := newTestServer(t, reviewServer(tokens, docs, issues, approvals))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	issues.AssertNotCalled(t, "Update", mock.Anything)
}

func TestDocumentsApprove_NonPanelManagerForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager) // manager but NOT on the panel
	docs := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	approvals := &mockDocumentApprovalRepo{}

	docs.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	issues.On("FindByID", uint(7)).Return(panelIssue(99, 2, false), nil)
	issues.On("GetReviewers", uint(7)).Return(usersWithIDs(5, 6, 7), nil)

	srv := newTestServer(t, reviewServer(tokens, docs, issues, approvals))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/approve", nil))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusForbidden, res.StatusCode)
	approvals.AssertNotCalled(t, "Create", mock.Anything)
}

func TestDocumentsReject_PanelReviewerVeto(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 5, models.RoleEmployee)
	docs := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	approvals := &mockDocumentApprovalRepo{}

	docs.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	docs.On("Update", mock.AnythingOfType("*models.Document")).Return(nil).Run(func(a mock.Arguments) {
		assert.Equal(t, models.DocumentStatusRejected, a.Get(0).(*models.Document).Status)
	})
	issues.On("FindByID", uint(7)).Return(panelIssue(99, 2, false), nil)
	issues.On("GetReviewers", uint(7)).Return(usersWithIDs(5, 6, 7), nil)
	approvals.On("ListByDocument", uint(42)).Return([]models.DocumentApproval{}, nil)

	srv := newTestServer(t, reviewServer(tokens, docs, issues, approvals))
	defer srv.Close()

	body := strings.NewReader(`{"note":"no good"}`)
	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/reject", body))
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	issues.AssertNotCalled(t, "Update", mock.Anything) // reject never resolves the issue
}

func TestDocumentsReject_NonPanelForbidden(t *testing.T) {
	tokens := &mockTokens{}
	stubTokenAccept(tokens, 1, models.RoleManager)
	docs := &mockDocumentRepo{}
	issues := &mockIssueRepo{}
	approvals := &mockDocumentApprovalRepo{}

	docs.On("FindByID", uint(42)).Return(&models.Document{
		Model: gormModelID(42), UploaderID: 1, IssueID: 7, Status: models.DocumentStatusInReview,
	}, nil)
	issues.On("FindByID", uint(7)).Return(panelIssue(99, 2, false), nil)
	issues.On("GetReviewers", uint(7)).Return(usersWithIDs(5, 6, 7), nil)

	srv := newTestServer(t, reviewServer(tokens, docs, issues, approvals))
	defer srv.Close()

	res, err := http.DefaultClient.Do(authedRequest(t, http.MethodPost, srv.URL+"/documents/42/reject", strings.NewReader(`{}`)))
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}
