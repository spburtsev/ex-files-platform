package models

import "gorm.io/gorm"

type IssueReviewer struct {
	gorm.Model
	IssueID uint  `gorm:"not null;uniqueIndex:idx_issue_reviewer"`
	Issue   Issue `gorm:"foreignKey:IssueID"`
	UserID  uint  `gorm:"not null;uniqueIndex:idx_issue_reviewer"`
	User    User  `gorm:"foreignKey:UserID"`
}
