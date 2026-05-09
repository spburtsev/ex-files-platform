package handlers

import (
	"context"
	"time"

	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

const dashboardDueSoonWindow = 72 * time.Hour

// DashboardGet implements GET /dashboard.
func (s *Server) DashboardGet(ctx context.Context) (oapi.DashboardGetRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.DashboardGetUnauthorized{Error: "unauthorized"}, nil
	}
	sum, err := s.IssueRepo.DashboardSummary(uid, dashboardDueSoonWindow)
	if err != nil {
		logErr("dashboard.summary", err)
		return &oapi.DashboardGetInternalServerError{Error: "failed to load dashboard"}, nil
	}

	dueSoon := make([]oapi.Issue, len(sum.DueSoon))
	for i := range sum.DueSoon {
		dueSoon[i] = issueToOAPI(&sum.DueSoon[i])
	}
	overdue := make([]oapi.Issue, len(sum.Overdue))
	for i := range sum.Overdue {
		overdue[i] = issueToOAPI(&sum.Overdue[i])
	}
	recent := make([]oapi.Issue, len(sum.Recent))
	for i := range sum.Recent {
		recent[i] = issueWithActivityToOAPI(&sum.Recent[i])
	}

	return &oapi.DashboardResponse{
		AssignedOpenCount: sum.AssignedOpenCount,
		CreatedOpenCount:  sum.CreatedOpenCount,
		DueSoon:           dueSoon,
		Overdue:           overdue,
		Recent:            recent,
	}, nil
}

// issueWithActivityToOAPI is identical to issueToOAPI but additionally copies
// the aggregated activity timestamp into LastActivityAt so the dashboard's
// recent list can sort/display it.
func issueWithActivityToOAPI(iwa *services.IssueWithActivity) oapi.Issue {
	out := issueToOAPI(&iwa.Issue)
	out.LastActivityAt = oapi.NewOptDateTime(iwa.LastActivityAt)
	return out
}
