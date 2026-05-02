package models

import "time"

type PasswordResetToken struct {
	ID        uint      `gorm:"primarykey"`
	Token     string    `gorm:"uniqueIndex;not null"`
	UserID    uint      `gorm:"not null;index"`
	User      User      `gorm:"foreignKey:UserID"`
	ExpiresAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"default:false"`
	CreatedAt time.Time
}

func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
