package models

import (
	"time"

	"gorm.io/gorm"
)

type Issue struct {
	gorm.Model
	WorkspaceID   uint       `gorm:"not null;index"`
	Workspace     Workspace  `gorm:"foreignKey:WorkspaceID"`
	CreatorID     uint       `gorm:"not null;index"`
	Creator       User       `gorm:"foreignKey:CreatorID"`
	AssigneeID    uint       `gorm:"not null;index"`
	Assignee      User       `gorm:"foreignKey:AssigneeID"`
	Title         string     `gorm:"not null"`
	Description   string     `gorm:"type:text"`
	Deadline      *time.Time
	Resolved      bool `gorm:"default:false"`
	CommentsCount int  `gorm:"default:0"`
	VersionsCount int  `gorm:"default:0"`
}
