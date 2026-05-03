package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/handlers"
	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

// gormModelID constructs a gorm.Model populated with the given primary key and
// CreatedAt/UpdatedAt set to a stable value so JSON marshaling is deterministic.
func gormModelID(id uint) gorm.Model {
	t := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	return gorm.Model{ID: id, CreatedAt: t, UpdatedAt: t}
}

// --- harness ------------------------------------------------------------

func newTestServer(t *testing.T, s *handlers.Server) *httptest.Server {
	t.Helper()
	og, err := oapi.NewServer(s, s)
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.Handle("/", og)
	root := middleware.WithCookieJar(mux)
	return httptest.NewServer(root)
}

// authedRequest builds a request with a Bearer token attached. Use stubTokenAccept
// to register the token with the mock token service. JSON content-type is added
// when a body is provided.
func authedRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// stubTokenAccept programs the mock token service to accept "test-token" and
// return claims for the given user.
func stubTokenAccept(tokens *mockTokens, userID uint, role models.Role) {
	tokens.On("Validate", "test-token").Return(&models.Claims{
		UserID: userID,
		Email:  "user@example.com",
		Role:   role,
	}, nil).Maybe()
}

func findCookie(jar []*http.Cookie, name string) *http.Cookie {
	for _, c := range jar {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// --- mocks: identity ----------------------------------------------------

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) FindByEmail(email string) (*models.User, error) {
	a := m.Called(email)
	if u, ok := a.Get(0).(*models.User); ok {
		return u, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockUserRepo) FindByID(id uint) (*models.User, error) {
	a := m.Called(id)
	if u, ok := a.Get(0).(*models.User); ok {
		return u, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockUserRepo) Create(u *models.User) error {
	args := m.Called(u)
	if id, ok := args.Get(0).(uint); ok {
		u.ID = id
	}
	return args.Error(1)
}
func (m *mockUserRepo) ListAll() ([]models.User, error) {
	a := m.Called()
	return a.Get(0).([]models.User), a.Error(1)
}
func (m *mockUserRepo) UpdatePassword(userID uint, hash string) error {
	return m.Called(userID, hash).Error(0)
}

type mockTokens struct{ mock.Mock }

func (m *mockTokens) Issue(u *models.User) (string, error) {
	a := m.Called(u)
	return a.String(0), a.Error(1)
}
func (m *mockTokens) Validate(tok string) (*models.Claims, error) {
	a := m.Called(tok)
	if c, ok := a.Get(0).(*models.Claims); ok {
		return c, a.Error(1)
	}
	return nil, a.Error(1)
}

type stubHasher struct{}

func (stubHasher) Hash(p string) (string, error) { return "hashed:" + p, nil }
func (stubHasher) Compare(h, p string) error {
	if h == "hashed:"+p {
		return nil
	}
	return errors.New("mismatch")
}

// --- mocks: audit -------------------------------------------------------

type dummyAudit struct{}

func (dummyAudit) Append(*models.AuditEntry) error { return nil }
func (dummyAudit) List(_ services.AuditFilter, _, _ int) ([]models.AuditEntry, int64, error) {
	return nil, 0, nil
}

type mockAuditRepo struct{ mock.Mock }

func (m *mockAuditRepo) Append(e *models.AuditEntry) error {
	return m.Called(e).Error(0)
}
func (m *mockAuditRepo) List(f services.AuditFilter, limit, offset int) ([]models.AuditEntry, int64, error) {
	a := m.Called(f, limit, offset)
	return a.Get(0).([]models.AuditEntry), a.Get(1).(int64), a.Error(2)
}

// --- mocks: workspaces --------------------------------------------------

type mockWorkspaceRepo struct{ mock.Mock }

func (m *mockWorkspaceRepo) Create(ws *models.Workspace) error {
	args := m.Called(ws)
	if id, ok := args.Get(0).(uint); ok {
		ws.ID = id
	}
	return args.Error(1)
}
func (m *mockWorkspaceRepo) FindByID(id uint) (*models.Workspace, error) {
	a := m.Called(id)
	if w, ok := a.Get(0).(*models.Workspace); ok {
		return w, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockWorkspaceRepo) FindByManager(managerID uint, search string, status models.WorkspaceStatus, limit, offset int) ([]models.Workspace, int64, error) {
	a := m.Called(managerID, search, status, limit, offset)
	return a.Get(0).([]models.Workspace), a.Get(1).(int64), a.Error(2)
}
func (m *mockWorkspaceRepo) FindByMember(userID uint, search string, status models.WorkspaceStatus, limit, offset int) ([]models.Workspace, int64, error) {
	a := m.Called(userID, search, status, limit, offset)
	return a.Get(0).([]models.Workspace), a.Get(1).(int64), a.Error(2)
}
func (m *mockWorkspaceRepo) Update(ws *models.Workspace) error {
	return m.Called(ws).Error(0)
}
func (m *mockWorkspaceRepo) Delete(id uint) error {
	return m.Called(id).Error(0)
}
func (m *mockWorkspaceRepo) AddMember(member *models.WorkspaceMember) error {
	args := m.Called(member)
	if id, ok := args.Get(0).(uint); ok {
		member.ID = id
		member.CreatedAt = time.Now()
	}
	return args.Error(1)
}
func (m *mockWorkspaceRepo) RemoveMember(workspaceID, userID uint) error {
	return m.Called(workspaceID, userID).Error(0)
}
func (m *mockWorkspaceRepo) GetMembers(workspaceID uint) ([]models.User, error) {
	a := m.Called(workspaceID)
	return a.Get(0).([]models.User), a.Error(1)
}
func (m *mockWorkspaceRepo) GetAssignableUsers(workspaceID uint) ([]models.User, error) {
	a := m.Called(workspaceID)
	return a.Get(0).([]models.User), a.Error(1)
}

// --- mocks: issues ------------------------------------------------------

type mockIssueRepo struct{ mock.Mock }

func (m *mockIssueRepo) ListAll() ([]models.Issue, error) {
	a := m.Called()
	return a.Get(0).([]models.Issue), a.Error(1)
}
func (m *mockIssueRepo) ListByWorkspace(workspaceID uint, search string, resolved *bool, archived bool) ([]models.Issue, error) {
	a := m.Called(workspaceID, search, resolved, archived)
	return a.Get(0).([]models.Issue), a.Error(1)
}
func (m *mockIssueRepo) FindByID(id uint) (*models.Issue, error) {
	a := m.Called(id)
	if i, ok := a.Get(0).(*models.Issue); ok {
		return i, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockIssueRepo) Create(issue *models.Issue) error {
	args := m.Called(issue)
	if id, ok := args.Get(0).(uint); ok {
		issue.ID = id
	}
	return args.Error(1)
}
func (m *mockIssueRepo) Update(issue *models.Issue) error {
	return m.Called(issue).Error(0)
}

// --- mocks: documents ---------------------------------------------------

type mockDocumentRepo struct{ mock.Mock }

func (m *mockDocumentRepo) Create(d *models.Document) error {
	args := m.Called(d)
	if id, ok := args.Get(0).(uint); ok {
		d.ID = id
	}
	return args.Error(1)
}
func (m *mockDocumentRepo) FindByID(id uint) (*models.Document, error) {
	a := m.Called(id)
	if d, ok := a.Get(0).(*models.Document); ok {
		return d, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockDocumentRepo) FindByHash(hash string) (*models.Document, error) {
	a := m.Called(hash)
	if d, ok := a.Get(0).(*models.Document); ok {
		return d, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockDocumentRepo) FindByIssueAndHash(issueID uint, hash string) (*models.Document, error) {
	a := m.Called(issueID, hash)
	if d, ok := a.Get(0).(*models.Document); ok {
		return d, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockDocumentRepo) Update(d *models.Document) error {
	return m.Called(d).Error(0)
}
func (m *mockDocumentRepo) ListByIssue(issueID uint, search, status string, limit, offset int) ([]models.Document, int64, error) {
	a := m.Called(issueID, search, status, limit, offset)
	return a.Get(0).([]models.Document), a.Get(1).(int64), a.Error(2)
}
func (m *mockDocumentRepo) Delete(id uint) error {
	return m.Called(id).Error(0)
}
func (m *mockDocumentRepo) CreateVersion(v *models.DocumentVersion) error {
	args := m.Called(v)
	if id, ok := args.Get(0).(uint); ok {
		v.ID = id
	}
	return args.Error(1)
}
func (m *mockDocumentRepo) GetVersions(documentID uint) ([]models.DocumentVersion, error) {
	a := m.Called(documentID)
	return a.Get(0).([]models.DocumentVersion), a.Error(1)
}
func (m *mockDocumentRepo) GetVersion(id uint) (*models.DocumentVersion, error) {
	a := m.Called(id)
	if v, ok := a.Get(0).(*models.DocumentVersion); ok {
		return v, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockDocumentRepo) LatestVersionNumber(documentID uint) (int, error) {
	a := m.Called(documentID)
	return a.Int(0), a.Error(1)
}

// --- mocks: comments ----------------------------------------------------

type mockCommentRepo struct{ mock.Mock }

func (m *mockCommentRepo) Create(c *models.Comment) error {
	args := m.Called(c)
	if id, ok := args.Get(0).(uint); ok {
		c.ID = id
	}
	return args.Error(1)
}
func (m *mockCommentRepo) FindByID(id uint) (*models.Comment, error) {
	a := m.Called(id)
	if c, ok := a.Get(0).(*models.Comment); ok {
		return c, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockCommentRepo) ListByDocument(documentID uint) ([]models.Comment, error) {
	a := m.Called(documentID)
	return a.Get(0).([]models.Comment), a.Error(1)
}

// --- mocks: storage -----------------------------------------------------

type mockStorage struct{ mock.Mock }

func (m *mockStorage) Upload(ctx context.Context, key string, r io.Reader, size int64, ct string) error {
	return m.Called(ctx, key, r, size, ct).Error(0)
}
func (m *mockStorage) PresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	a := m.Called(ctx, key, expires)
	return a.String(0), a.Error(1)
}
func (m *mockStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	a := m.Called(ctx, key)
	if rc, ok := a.Get(0).(io.ReadCloser); ok {
		return rc, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
