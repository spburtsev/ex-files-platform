package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Comment struct {
	gorm.Model
	DocumentID uint              `gorm:"not null;index"`
	Document   Document          `gorm:"foreignKey:DocumentID"`
	AuthorID   uint              `gorm:"not null;index"`
	Author     User              `gorm:"foreignKey:AuthorID"`
	Body       string            `gorm:"type:text;not null"`
	Metadata   datatypes.JSONMap `gorm:"type:jsonb"`
}
