package services

import (
	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/models"
)

type GormAuditRepository struct {
	DB *gorm.DB
}

func (r *GormAuditRepository) Append(entry *models.AuditEntry) error {
	return r.DB.Create(entry).Error
}

func (r *GormAuditRepository) List(filter AuditFilter, limit, offset int) ([]models.AuditEntry, int64, error) {
	var entries []models.AuditEntry
	var total int64

	q := r.DB.Model(&models.AuditEntry{})

	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if filter.ActorID != nil {
		q = q.Where("actor_id = ?", *filter.ActorID)
	}
	if filter.TargetID != nil {
		q = q.Where("target_id = ?", *filter.TargetID)
	}
	if filter.TargetType != "" {
		q = q.Where("target_type = ?", filter.TargetType)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := q.Preload("Actor").Order("created_at DESC").Limit(limit).Offset(offset).Find(&entries).Error; err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}
