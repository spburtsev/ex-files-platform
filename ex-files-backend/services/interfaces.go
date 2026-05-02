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

type AuditFilter struct {
	Action     string
	ActorID    *uint
	TargetID   *uint
	TargetType string
	From       *time.Time
	To         *time.Time
}

type AuditRepository interface {
	Append(entry *models.AuditEntry) error
	List(filter AuditFilter, limit, offset int) ([]models.AuditEntry, int64, error)
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
	CreateVersion(version *models.DocumentVersion) error
	GetVersions(documentID uint) ([]models.DocumentVersion, error)
	GetVersion(id uint) (*models.DocumentVersion, error)
	LatestVersionNumber(documentID uint) (int, error)
}

type IssueRepository interface {
	ListAll() ([]models.Issue, error)
	ListByWorkspace(workspaceID uint) ([]models.Issue, error)
	FindByID(id uint) (*models.Issue, error)
	Create(issue *models.Issue) error
	Update(issue *models.Issue) error
}

type EmailService interface {
	Send(to, subject, body string) error
}

type CommentRepository interface {
	Create(comment *models.Comment) error
	FindByID(id uint) (*models.Comment, error)
	ListByDocument(documentID uint) ([]models.Comment, error)
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
	GetAssignableUsers(workspaceID uint) ([]models.User, error)
}
