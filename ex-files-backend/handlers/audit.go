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

// List returns the audit log with optional filters.
// @Summary      List audit entries
// @Tags         audit
// @Produce      application/x-protobuf
// @Param        action       query  string  false  "Filter by action"
// @Param        target_type  query  string  false  "Filter by target type"
// @Param        actor_id     query  int     false  "Filter by actor ID"
// @Param        target_id    query  int     false  "Filter by target ID"
// @Param        from         query  string  false  "From date (RFC3339)"
// @Param        to           query  string  false  "To date (RFC3339)"
// @Param        page         query  int     false  "Page number"     default(1)
// @Param        per_page     query  int     false  "Items per page"  default(20)
// @Success      200  {object}  swagGetAuditLogResponse  "Protobuf: audit.v1.GetAuditLogResponse"
// @Header       200  {int}     X-Total-Count
// @Header       200  {int}     X-Total-Pages
// @Header       200  {int}     X-Page
// @Header       200  {int}     X-Per-Page
// @Security     BearerAuth || CookieAuth
// @Router       /audit [get]
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

	protobufResponse(c, http.StatusOK, &auditv1.GetAuditLogResponse{
		Entries: pbEntries,
	})
}
