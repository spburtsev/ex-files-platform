package models

import "gorm.io/gorm"

type WorkspaceStatus string

const (
	WorkspaceStatusActive   WorkspaceStatus = "active"
	WorkspaceStatusArchived WorkspaceStatus = "archived"
)

type Workspace struct {
	gorm.Model
	Name      string          `gorm:"not null"`
	Status    WorkspaceStatus `gorm:"not null;default:'active'"`
	ManagerID uint            `gorm:"not null"`
	Manager   User            `gorm:"foreignKey:ManagerID"`
}

func (w *Workspace) IsOwnedBy(userID uint) bool {
	return w.ManagerID == userID
}

type WorkspaceMember struct {
	gorm.Model
	WorkspaceID uint      `gorm:"not null;uniqueIndex:idx_ws_user"`
	Workspace   Workspace `gorm:"foreignKey:WorkspaceID"`
	UserID      uint      `gorm:"not null;uniqueIndex:idx_ws_user"`
	User        User      `gorm:"foreignKey:UserID"`
}
