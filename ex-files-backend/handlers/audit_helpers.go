package handlers

import (
	"log"

	"gorm.io/datatypes"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

func logAudit(repo services.AuditRepository, action models.AuditAction, actorID uint, targetID *uint, targetType string, metadata map[string]any) {
	if repo == nil {
		return
	}
	entry := &models.AuditEntry{
		Action:     action,
		ActorID:    actorID,
		TargetID:   targetID,
		TargetType: targetType,
		Metadata:   datatypes.JSONMap(metadata),
	}
	if err := repo.Append(entry); err != nil {
		log.Printf("audit: failed to log %s: %v", action, err)
	}
}

func uintPtr(v uint) *uint {
	return &v
}
