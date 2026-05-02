package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-faster/jx"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

func auditEntryToOAPI(e *models.AuditEntry) oapi.AuditEntry {
	out := oapi.AuditEntry{
		ID:         formatID(e.ID),
		Action:     string(e.Action),
		ActorId:    formatID(e.ActorID),
		ActorName:  e.Actor.Name,
		TargetType: e.TargetType,
		CreatedAt:  e.CreatedAt,
	}
	if e.TargetID != nil {
		out.TargetId = oapi.NewOptNilString(formatID(*e.TargetID))
	}
	if len(e.Metadata) > 0 {
		md := oapi.AuditEntryMetadata{}
		for k, v := range e.Metadata {
			b, err := json.Marshal(v)
			if err == nil {
				md[k] = jx.Raw(b)
			}
		}
		out.Metadata = oapi.NewOptNilAuditEntryMetadata(md)
	}
	return out
}

// AuditList implements GET /audit.
func (s *Server) AuditList(ctx context.Context, params oapi.AuditListParams) (oapi.AuditListRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.AuditListUnauthorized{Error: "unauthorized"}, nil
	}

	page, perPage, offset := resolvePagination(params.Page, params.PerPage)

	filter := services.AuditFilter{
		Action:     params.Action.Or(""),
		TargetType: params.TargetType.Or(""),
	}
	if v, ok := params.ActorId.Get(); ok {
		if id, ok := parseUintID(v); ok {
			filter.ActorID = &id
		}
	}
	if v, ok := params.TargetId.Get(); ok {
		if id, ok := parseUintID(v); ok {
			filter.TargetID = &id
		}
	}
	if v, ok := params.From.Get(); ok {
		t := v
		filter.From = &t
	}
	if v, ok := params.To.Get(); ok {
		t := v
		filter.To = &t
	}

	entries, total, err := s.Audit.List(filter, perPage, offset)
	if err != nil {
		logErr("audit.list", err)
		return &oapi.AuditListInternalServerError{Error: "failed to fetch audit log"}, nil
	}

	out := make([]oapi.AuditEntry, len(entries))
	for i := range entries {
		out[i] = auditEntryToOAPI(&entries[i])
	}

	return &oapi.GetAuditLogResponseHeaders{
		XPage:       optInt32(page),
		XPerPage:    optInt32(perPage),
		XTotalCount: optInt64(total),
		XTotalPages: optInt32(totalPages(total, perPage)),
		Response:    oapi.GetAuditLogResponse{Entries: out},
	}, nil
}

// AuditStats implements GET /audit/stats.
func (s *Server) AuditStats(ctx context.Context) (oapi.AuditStatsRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.AuditStatsUnauthorized{Error: "unauthorized"}, nil
	}
	if s.DB == nil {
		return &oapi.AuditStatsInternalServerError{Error: "stats not available"}, nil
	}

	type rowAction struct {
		Action string
		Count  int64
	}
	var actions []rowAction
	s.DB.Model(&models.AuditEntry{}).
		Select("action, count(*) as count").
		Group("action").
		Order("count DESC").
		Scan(&actions)

	type rowDaily struct {
		Date  string
		Count int64
	}
	var daily []rowDaily
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	s.DB.Model(&models.AuditEntry{}).
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at >= ?", thirtyDaysAgo).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&daily)

	type rowStatus struct {
		Status string
		Count  int64
	}
	var statuses []rowStatus
	s.DB.Model(&models.Document{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statuses)

	type rowActor struct {
		ActorID   uint
		ActorName string
		Count     int64
	}
	var actors []rowActor
	s.DB.Model(&models.AuditEntry{}).
		Select("audit_entries.actor_id, users.name as actor_name, count(*) as count").
		Joins("LEFT JOIN users ON users.id = audit_entries.actor_id").
		Group("audit_entries.actor_id, users.name").
		Order("count DESC").
		Limit(10).
		Scan(&actors)

	resp := oapi.AuditStatsResponse{
		ActionsByType:     make([]oapi.AuditActionCount, len(actions)),
		DailyActivity:     make([]oapi.AuditDailyActivity, len(daily)),
		DocumentsByStatus: make([]oapi.DocumentStatusCount, len(statuses)),
		TopActors:         make([]oapi.AuditTopActor, len(actors)),
	}
	for i, a := range actions {
		resp.ActionsByType[i] = oapi.AuditActionCount{Action: a.Action, Count: a.Count}
	}
	for i, d := range daily {
		resp.DailyActivity[i] = oapi.AuditDailyActivity{Date: d.Date, Count: d.Count}
	}
	for i, st := range statuses {
		resp.DocumentsByStatus[i] = oapi.DocumentStatusCount{Status: st.Status, Count: st.Count}
	}
	for i, a := range actors {
		resp.TopActors[i] = oapi.AuditTopActor{
			ActorId:   formatID(a.ActorID),
			ActorName: a.ActorName,
			Count:     a.Count,
		}
	}
	return &resp, nil
}
