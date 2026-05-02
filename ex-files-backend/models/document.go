package models

import "gorm.io/gorm"

type DocumentStatus string

const (
	DocumentStatusPending          DocumentStatus = "pending"
	DocumentStatusInReview         DocumentStatus = "in_review"
	DocumentStatusApproved         DocumentStatus = "approved"
	DocumentStatusRejected         DocumentStatus = "rejected"
	DocumentStatusChangesRequested DocumentStatus = "changes_requested"
)

// validTransitions defines allowed status transitions.
//
// Reviewers (managers / assigned reviewers) may short-circuit from pending or
// changes_requested directly to a terminal review status without an explicit
// submit step. Caller-role authorisation is still enforced at the handler
// layer; this map only declares what the document state machine permits.
var validTransitions = map[DocumentStatus][]DocumentStatus{
	DocumentStatusPending: {
		DocumentStatusInReview,
		DocumentStatusApproved,
		DocumentStatusRejected,
		DocumentStatusChangesRequested,
	},
	DocumentStatusInReview: {
		DocumentStatusApproved,
		DocumentStatusRejected,
		DocumentStatusChangesRequested,
	},
	DocumentStatusChangesRequested: {
		DocumentStatusInReview,
		DocumentStatusApproved,
		DocumentStatusRejected,
	},
}

// CanTransitionTo reports whether transitioning from the current status to next is valid.
func (d *Document) CanTransitionTo(next DocumentStatus) bool {
	allowed, ok := validTransitions[d.Status]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

type Document struct {
	gorm.Model
	Name         string         `gorm:"not null"`
	MimeType     string         `gorm:"not null"`
	Size         int64          `gorm:"not null"`
	Hash         string         `gorm:"type:varchar(64);not null;index"`
	Status       DocumentStatus `gorm:"type:varchar(30);default:pending;not null"`
	UploaderID   uint           `gorm:"not null;index"`
	Uploader     User           `gorm:"foreignKey:UploaderID"`
	IssueID      uint           `gorm:"not null;index"`
	Issue        Issue          `gorm:"foreignKey:IssueID"`
	ReviewerID   *uint          `gorm:"index"`
	Reviewer     User           `gorm:"foreignKey:ReviewerID"`
	ReviewerNote string         `gorm:"type:text"`
}

type DocumentVersion struct {
	gorm.Model
	DocumentID uint     `gorm:"not null;index"`
	Document   Document `gorm:"foreignKey:DocumentID"`
	Version    int      `gorm:"not null"`
	Hash       string   `gorm:"type:varchar(64);not null"`
	Size       int64    `gorm:"not null"`
	StorageKey string   `gorm:"not null"`
	UploaderID uint     `gorm:"not null"`
	Uploader   User     `gorm:"foreignKey:UploaderID"`
}
