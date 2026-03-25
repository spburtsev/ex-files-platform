package models

import "gorm.io/gorm"

type Role string

const (
	RoleRoot     Role = "root"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
)

func (r Role) CanManageWorkspaces() bool {
	return r == RoleManager || r == RoleRoot
}

type User struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"`
	Name         string
	AvatarURL    string
	PasswordHash string `gorm:"not null"`
	Role         Role   `gorm:"type:varchar(20);default:employee"`
}
