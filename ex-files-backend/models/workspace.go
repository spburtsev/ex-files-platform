package models

import "gorm.io/gorm"

type Workspace struct {
	gorm.Model
	Name      string `gorm:"not null"`
	ManagerID uint   `gorm:"not null"`
	Manager   User   `gorm:"foreignKey:ManagerID"`
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
