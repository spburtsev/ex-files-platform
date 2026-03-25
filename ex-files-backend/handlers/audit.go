package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	auditv1 "github.com/spburtsev/ex-files-backend/gen/audit/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type AuditHandler struct {
	Repo services.AuditRepository
}

func auditEntryToProto(e *models.AuditEntry) *auditv1.AuditEntry {
	entry := &auditv1.AuditEntry{
		Id:         uint64(e.ID),
		Action:     string(e.Action),
		ActorId:    uint64(e.ActorID),
		ActorName:  e.Actor.Name,
		TargetType: e.TargetType,
		CreatedAt:  timestamppb.New(e.CreatedAt),
	}

	if e.TargetID != nil {
		tid := uint64(*e.TargetID)
		entry.TargetId = &tid
	}

	if e.Metadata != nil {
		if s, err := structpb.NewStruct(e.Metadata); err == nil {
			entry.Metadata = s
		}
	}

	return entry
}

func (h *AuditHandler) List(c *gin.Context) {
	page, perPage := parsePagination(c)
	offset := (page - 1) * perPage

	filter := services.AuditFilter{
		Action:     c.Query("action"),
		TargetType: c.Query("target_type"),
	}

	if v := c.Query("actor_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			uid := uint(id)
			filter.ActorID = &uid
		}
	}

	if v := c.Query("target_id"); v != "" {
		if id, err := strconv.ParseUint(v, 10, 64); err == nil {
			uid := uint(id)
			filter.TargetID = &uid
		}
	}

	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.From = &t
		}
	}

	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.To = &t
		}
	}

	entries, total, err := h.Repo.List(filter, perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit log"})
		return
	}

	setPaginationHeaders(c, page, perPage, total)

	pbEntries := make([]*auditv1.AuditEntry, len(entries))
	for i := range entries {
		pbEntries[i] = auditEntryToProto(&entries[i])
	}

	c.JSON(http.StatusOK, &auditv1.GetAuditLogResponse{
		Entries: pbEntries,
	})
}
