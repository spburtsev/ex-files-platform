package models

import "gorm.io/gorm"

// DocumentApproval records one reviewer's approval of one document version.
// Approvals are per-document, so a resubmit/new version starts a fresh tally.
// The composite unique index guards against a reviewer being counted twice;
// CreatedAt (from gorm.Model) is the "approved at" timestamp shown in the UI.
type DocumentApproval struct {
	gorm.Model
	DocumentID uint     `gorm:"not null;uniqueIndex:idx_doc_reviewer"`
	Document   Document `gorm:"foreignKey:DocumentID"`
	ReviewerID uint     `gorm:"not null;uniqueIndex:idx_doc_reviewer"`
	Reviewer   User     `gorm:"foreignKey:ReviewerID"`
}
