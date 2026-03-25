package services

import (
	"context"
	"io"
	"time"

	"github.com/spburtsev/ex-files-backend/models"
)

type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	Create(user *models.User) error
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
	Delete(ctx context.Context, key string) error
}

type DocumentRepository interface {
	Create(doc *models.Document) error
	FindByID(id uint) (*models.Document, error)
	ListByWorkspace(workspaceID uint, search, status string, limit, offset int) ([]models.Document, int64, error)
	Delete(id uint) error
	CreateVersion(version *models.DocumentVersion) error
	GetVersions(documentID uint) ([]models.DocumentVersion, error)
	GetVersion(id uint) (*models.DocumentVersion, error)
	LatestVersionNumber(documentID uint) (int, error)
}

type WorkspaceRepository interface {
	Create(workspace *models.Workspace) error
	FindByID(id uint) (*models.Workspace, error)
	FindByManager(managerID uint, limit, offset int) ([]models.Workspace, int64, error)
	FindByMember(userID uint, limit, offset int) ([]models.Workspace, int64, error)
	Update(workspace *models.Workspace) error
	Delete(id uint) error
	AddMember(member *models.WorkspaceMember) error
	RemoveMember(workspaceID, userID uint) error
	GetMembers(workspaceID uint) ([]models.User, error)
}
