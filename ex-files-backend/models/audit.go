package models

import (
	"time"

	"gorm.io/datatypes"
)

type AuditAction string

const (
	AuditActionUserRegistered    AuditAction = "user.registered"
	AuditActionUserLoggedIn      AuditAction = "user.logged_in"
	AuditActionWorkspaceCreated  AuditAction = "workspace.created"
	AuditActionWorkspaceUpdated  AuditAction = "workspace.updated"
	AuditActionWorkspaceDeleted  AuditAction = "workspace.deleted"
	AuditActionMemberAdded       AuditAction = "workspace.member_added"
	AuditActionMemberRemoved     AuditAction = "workspace.member_removed"
	AuditActionDocumentUploaded  AuditAction = "document.uploaded"
	AuditActionDocumentApproved  AuditAction = "document.approved"
	AuditActionDocumentRejected  AuditAction = "document.rejected"
	AuditActionVersionCreated    AuditAction = "document.version_created"
	AuditActionCommentAdded      AuditAction = "document.comment_added"
	AuditActionRoleChanged       AuditAction = "user.role_changed"
)

// AuditEntry is append-only. It has no UpdatedAt or DeletedAt fields.
type AuditEntry struct {
	ID         uint              `gorm:"primarykey"`
	CreatedAt  time.Time         `gorm:"index;not null"`
	Action     AuditAction       `gorm:"type:varchar(50);not null;index"`
	ActorID    uint              `gorm:"not null;index"`
	Actor      User              `gorm:"foreignKey:ActorID"`
	TargetID   *uint             `gorm:"index"`
	TargetType string            `gorm:"type:varchar(30)"`
	Metadata   datatypes.JSONMap `gorm:"type:jsonb"`
}
