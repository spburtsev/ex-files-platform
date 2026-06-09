package services

import (
	"context"
	"io"
	"time"

	"github.com/spburtsev/ex-files-backend/models"
)

type CacheService interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
}

type ResetTokenStore interface {
	StoreResetToken(token string, userID uint, ttl time.Duration) error
	GetResetTokenUserID(token string) (uint, error)
	DeleteResetToken(token string) error
}

type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	Create(user *models.User) error
	ListAll() ([]models.User, error)
	UpdatePassword(userID uint, passwordHash string) error
}

type TokenService interface {
	Issue(user *models.User) (string, error)
	Validate(tokenStr string) (*models.Claims, error)
}

type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type StorageService interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	PresignedURL(ctx context.Context, key string, expires time.Duration) (string, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type DocumentRepository interface {
	Create(doc *models.Document) error
	FindByID(id uint) (*models.Document, error)
	FindByHash(hash string) (*models.Document, error)
	FindByIssueAndHash(issueID uint, hash string) (*models.Document, error)
	Update(doc *models.Document) error
	ListByIssue(issueID uint, search, status string, limit, offset int) ([]models.Document, int64, error)
	Delete(id uint) error
}

type DocumentApprovalRepository interface {
	Create(approval *models.DocumentApproval) error
	ListByDocument(documentID uint) ([]models.DocumentApproval, error)
	ListByDocumentIDs(documentIDs []uint) ([]models.DocumentApproval, error)
	CountByReviewers(documentID uint, reviewerIDs []uint) (int64, error)
	DeleteByDocument(documentID uint) error
}

type IssueRepository interface {
	ListAll() ([]models.Issue, error)
	ListByWorkspace(workspaceID uint, search string, resolved *bool, archived bool) ([]models.Issue, error)
	ListUnresolvedWithDeadline() ([]models.Issue, error)
	ListMyCurrentIssues(userID uint, search string, limit, offset int) ([]models.Issue, int64, error)
	FindByID(id uint) (*models.Issue, error)
	Create(issue *models.Issue) error
	Update(issue *models.Issue) error
	DashboardSummary(userID uint, dueSoonWindow time.Duration) (DashboardSummary, error)
	GetReviewers(issueID uint) ([]models.User, error)
	SetReviewers(issueID uint, userIDs []uint) error
}

type IssueWithActivity struct {
	models.Issue
	LastActivityAt time.Time
}

type DashboardSummary struct {
	AssignedOpenCount int64
	CreatedOpenCount  int64
	DueSoon           []models.Issue
	Overdue           []models.Issue
	Recent            []IssueWithActivity
}

type EmailService interface {
	Send(to, subject, body string) error
}

type CommentRepository interface {
	Create(comment *models.Comment) error
	FindByID(id uint) (*models.Comment, error)
	ListByDocument(documentID uint) ([]models.Comment, error)
	Delete(id uint) error
}

type WorkspaceRepository interface {
	Create(workspace *models.Workspace) error
	FindByID(id uint) (*models.Workspace, error)
	FindByManager(managerID uint, search string, status models.WorkspaceStatus, limit, offset int) ([]models.Workspace, int64, error)
	FindByMember(userID uint, search string, status models.WorkspaceStatus, limit, offset int) ([]models.Workspace, int64, error)
	Update(workspace *models.Workspace) error
	Delete(id uint) error
	AddMember(member *models.WorkspaceMember) error
	RemoveMember(workspaceID, userID uint) error
	GetMembers(workspaceID uint) ([]models.User, error)
	IsMember(workspaceID, userID uint) (bool, error)
	GetAssignableUsers(workspaceID uint) ([]models.User, error)
}
