package handlers

import (
	"context"
	"log/slog"

	"github.com/spburtsev/ex-files-backend/logging"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func issueToOAPI(i *models.Issue) oapi.Issue {
	out := oapi.Issue{
		ID:                formatID(i.ID),
		WorkspaceId:       formatID(i.WorkspaceID),
		CreatorId:         formatID(i.CreatorID),
		AssigneeId:        formatID(i.AssigneeID),
		Title:             i.Title,
		Description:       i.Description,
		Resolved:          i.Resolved,
		Archived:          oapi.NewOptBool(i.Archived),
		CommentsCount:     int32(i.CommentsCount),
		VersionsCount:     int32(i.VersionsCount),
		RequiredApprovals: oapi.NewOptInt32(int32(requiredApprovalsOf(i))),
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
	if i.Deadline != nil {
		out.Deadline = oapi.NewOptNilDateTime(*i.Deadline)
	}
	if len(i.Reviewers) > 0 {
		revs := make([]oapi.ReviewerSummary, 0, len(i.Reviewers))
		for j := range i.Reviewers {
			revs = append(revs, oapi.ReviewerSummary{
				ID:   formatID(i.Reviewers[j].UserID),
				Name: i.Reviewers[j].User.Name,
			})
		}
		out.Reviewers = revs
	}
	return out
}

func (s *Server) resolveReviewerIDs(ws *models.Workspace, raw []string) ([]uint, bool) {
	if len(raw) == 0 {
		return nil, true
	}
	members, err := s.WorkspaceRepo.GetMembers(ws.ID)
	if err != nil {
		logErr("issues.reviewers.members", err)
		return nil, false
	}
	allowed := make(map[uint]bool, len(members)+1)
	for i := range members {
		allowed[members[i].ID] = true
	}
	allowed[ws.ManagerID] = true
	seen := make(map[uint]bool, len(raw))
	ids := make([]uint, 0, len(raw))
	for _, r := range raw {
		id, ok := parseUintID(r)
		if !ok || !allowed[id] {
			return nil, false
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, true
}

func clampRequiredApprovals(n, panelSize int) int {
	if panelSize == 0 {
		return 1
	}
	if n < 1 {
		return 1
	}
	if n > panelSize {
		return panelSize
	}
	return n
}

// IssuesGet implements GET /issues/{id}.
func (s *Server) IssuesGet(ctx context.Context, params oapi.IssuesGetParams) (oapi.IssuesGetRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.IssuesGetUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.IssuesGetNotFound{Error: "issue not found"}, nil
	}
	issue, err := s.IssueRepo.FindByID(id)
	if err != nil {
		return &oapi.IssuesGetNotFound{Error: "issue not found"}, nil
	}
	if allowed, err := s.canViewIssue(issue, uid, role); err != nil {
		logErr("issues.get.authz", err)
		return &oapi.IssuesGetInternalServerError{Error: "failed to check access"}, nil
	} else if !allowed {
		return &oapi.IssuesGetNotFound{Error: "issue not found"}, nil
	}
	return &oapi.GetIssueResponse{
		Issue: issueToOAPI(issue),
		User:  userToOAPI(&issue.Assignee),
	}, nil
}

// IssuesListByWorkspace implements GET /workspaces/{id}/issues.
func (s *Server) IssuesListByWorkspace(ctx context.Context, params oapi.IssuesListByWorkspaceParams) (oapi.IssuesListByWorkspaceRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.IssuesListByWorkspaceUnauthorized{Error: "unauthorized"}, nil
	}
	wsID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.IssuesListByWorkspaceNotFound{Error: "workspace not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(wsID)
	if err != nil {
		return &oapi.IssuesListByWorkspaceNotFound{Error: "workspace not found"}, nil
	}
	if allowed, err := s.canViewWorkspace(ws, uid, role); err != nil {
		logErr("issues.list.authz", err)
		return &oapi.IssuesListByWorkspaceInternalServerError{Error: "failed to check access"}, nil
	} else if !allowed {
		return &oapi.IssuesListByWorkspaceNotFound{Error: "workspace not found"}, nil
	}
	search := params.Search.Or("")
	var resolved *bool
	switch params.Status.Or(oapi.IssuesListByWorkspaceStatusAll) {
	case oapi.IssuesListByWorkspaceStatusOpen:
		f := false
		resolved = &f
	case oapi.IssuesListByWorkspaceStatusResolved:
		t := true
		resolved = &t
	}
	archived := params.Archived.Or(false)
	issues, err := s.IssueRepo.ListByWorkspace(wsID, search, resolved, archived)
	if err != nil {
		logErr("issues.list", err)
		return &oapi.IssuesListByWorkspaceInternalServerError{Error: "failed to list issues"}, nil
	}
	out := make([]oapi.Issue, len(issues))
	for i := range issues {
		out[i] = issueToOAPI(&issues[i])
	}
	return &oapi.GetIssuesResponse{Issues: out}, nil
}

// IssuesUpdateAssignee implements PUT /issues/{id}/assignee.
func (s *Server) IssuesUpdateAssignee(ctx context.Context, req *oapi.UpdateAssigneeRequest, params oapi.IssuesUpdateAssigneeParams) (oapi.IssuesUpdateAssigneeRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.IssuesUpdateAssigneeUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.IssuesUpdateAssigneeNotFound{Error: "issue not found"}, nil
	}
	issue, err := s.IssueRepo.FindByID(id)
	if err != nil {
		return &oapi.IssuesUpdateAssigneeNotFound{Error: "issue not found"}, nil
	}
	if issue.CreatorID != uid && role != models.RoleRoot {
		ws, err := s.WorkspaceRepo.FindByID(issue.WorkspaceID)
		if err != nil {
			logErr("issues.update_assignee.workspace", err)
			return &oapi.IssuesUpdateAssigneeInternalServerError{Error: "failed to load workspace"}, nil
		}
		if !ws.IsOwnedBy(uid) {
			return &oapi.IssuesUpdateAssigneeForbidden{Error: "only the workspace manager or the issue creator may change the assignee"}, nil
		}
	}
	if issue.Resolved {
		return &oapi.IssuesUpdateAssigneeUnprocessableEntity{Error: "cannot change assignee of a resolved issue"}, nil
	}
	assigneeID, ok := parseUintID(req.AssigneeId)
	if !ok {
		return &oapi.IssuesUpdateAssigneeBadRequest{Error: "invalid assigneeId"}, nil
	}
	if _, err := s.UserRepo.FindByID(assigneeID); err != nil {
		return &oapi.IssuesUpdateAssigneeBadRequest{Error: "assignee not found"}, nil
	}
	issue.AssigneeID = assigneeID
	if err := s.IssueRepo.Update(issue); err != nil {
		logErr("issues.update_assignee", err)
		return &oapi.IssuesUpdateAssigneeInternalServerError{Error: "failed to update issue"}, nil
	}
	refreshed, err := s.IssueRepo.FindByID(id)
	if err != nil {
		logErr("issues.update_assignee.refetch", err)
		return &oapi.IssuesUpdateAssigneeInternalServerError{Error: "failed to load updated issue"}, nil
	}
	return &oapi.GetIssueResponse{
		Issue: issueToOAPI(refreshed),
		User:  userToOAPI(&refreshed.Assignee),
	}, nil
}

// IssuesCreate implements POST /workspaces/{id}/issues.
func (s *Server) IssuesCreate(ctx context.Context, req *oapi.CreateIssueRequest, params oapi.IssuesCreateParams) (oapi.IssuesCreateRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.IssuesCreateUnauthorized{Error: "unauthorized"}, nil
	}
	if !role.CanManageWorkspaces() {
		return &oapi.IssuesCreateForbidden{Error: "only managers may create issues"}, nil
	}
	wsID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.IssuesCreateBadRequest{Error: "invalid workspace id"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(wsID)
	if err != nil {
		return &oapi.IssuesCreateBadRequest{Error: "workspace not found"}, nil
	}
	if role != models.RoleRoot && !ws.IsOwnedBy(uid) {
		return &oapi.IssuesCreateForbidden{Error: "only the workspace manager may create issues in this workspace"}, nil
	}
	assigneeID, ok := parseUintID(req.AssigneeId)
	if !ok {
		return &oapi.IssuesCreateBadRequest{Error: "invalid assigneeId"}, nil
	}

	var reviewerIDs []uint
	if len(req.ReviewerIds) > 0 {
		ids, ok := s.resolveReviewerIDs(ws, req.ReviewerIds)
		if !ok {
			return &oapi.IssuesCreateBadRequest{Error: "invalid reviewerIds"}, nil
		}
		reviewerIDs = ids
	}

	issue := models.Issue{
		WorkspaceID:       wsID,
		CreatorID:         uid,
		AssigneeID:        assigneeID,
		Title:             req.Title,
		Description:       req.Description.Or(""),
		RequiredApprovals: clampRequiredApprovals(int(req.RequiredApprovals.Or(1)), len(reviewerIDs)),
	}
	if d, ok := req.Deadline.Get(); ok {
		issue.Deadline = &d
	}
	if err := s.IssueRepo.Create(&issue); err != nil {
		logErr("issues.create", err)
		return &oapi.IssuesCreateInternalServerError{Error: "failed to create issue"}, nil
	}
	if len(reviewerIDs) > 0 {
		if err := s.IssueRepo.SetReviewers(issue.ID, reviewerIDs); err != nil {
			logErr("issues.create.reviewers", err)
			return &oapi.IssuesCreateInternalServerError{Error: "failed to set reviewers"}, nil
		}
		if refreshed, err := s.IssueRepo.FindByID(issue.ID); err == nil {
			issue = *refreshed
		}
	}
	return &oapi.CreateIssueResponse{Issue: issueToOAPI(&issue)}, nil
}

func (s *Server) IssuesUpdateReviewConfig(ctx context.Context, req *oapi.UpdateReviewConfigRequest, params oapi.IssuesUpdateReviewConfigParams) (oapi.IssuesUpdateReviewConfigRes, error) {
	uid, _, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.IssuesUpdateReviewConfigUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.IssuesUpdateReviewConfigNotFound{Error: "issue not found"}, nil
	}
	issue, err := s.IssueRepo.FindByID(id)
	if err != nil {
		return &oapi.IssuesUpdateReviewConfigNotFound{Error: "issue not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(issue.WorkspaceID)
	if err != nil {
		logErr("issues.review_config.workspace", err)
		return &oapi.IssuesUpdateReviewConfigInternalServerError{Error: "failed to load workspace"}, nil
	}
	if issue.CreatorID != uid && !ws.IsOwnedBy(uid) {
		return &oapi.IssuesUpdateReviewConfigForbidden{Error: "only the issue creator or workspace owner may configure reviewers"}, nil
	}
	if issue.Resolved {
		return &oapi.IssuesUpdateReviewConfigUnprocessableEntity{Error: "cannot configure reviewers of a resolved issue"}, nil
	}
	reviewerIDs, ok := s.resolveReviewerIDs(ws, req.ReviewerIds)
	if !ok {
		return &oapi.IssuesUpdateReviewConfigBadRequest{Error: "invalid reviewerIds"}, nil
	}
	n := int(req.RequiredApprovals)
	if len(reviewerIDs) > 0 && (n < 1 || n > len(reviewerIDs)) {
		return &oapi.IssuesUpdateReviewConfigUnprocessableEntity{Error: "requiredApprovals must be between 1 and the number of reviewers"}, nil
	}
	if err := s.IssueRepo.SetReviewers(id, reviewerIDs); err != nil {
		logErr("issues.review_config.set_reviewers", err)
		return &oapi.IssuesUpdateReviewConfigInternalServerError{Error: "failed to update reviewers"}, nil
	}
	issue.RequiredApprovals = clampRequiredApprovals(n, len(reviewerIDs))
	if err := s.IssueRepo.Update(issue); err != nil {
		logErr("issues.review_config.update", err)
		return &oapi.IssuesUpdateReviewConfigInternalServerError{Error: "failed to update issue"}, nil
	}
	logging.Audit(ctx, "issue.review_config_updated", uid,
		slog.String("target_type", "issue"),
		slog.Uint64("target_id", uint64(id)),
		slog.Int("reviewer_count", len(reviewerIDs)),
		slog.Int("required_approvals", issue.RequiredApprovals),
	)
	refreshed, err := s.IssueRepo.FindByID(id)
	if err != nil {
		logErr("issues.review_config.refetch", err)
		return &oapi.IssuesUpdateReviewConfigInternalServerError{Error: "failed to load updated issue"}, nil
	}
	return &oapi.GetIssueResponse{
		Issue: issueToOAPI(refreshed),
		User:  userToOAPI(&refreshed.Assignee),
	}, nil
}

// IssuesArchive implements PUT /issues/{id}/archive.
func (s *Server) IssuesArchive(ctx context.Context, req *oapi.ArchiveIssueRequest, params oapi.IssuesArchiveParams) (oapi.IssuesArchiveRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.IssuesArchiveUnauthorized{Error: "unauthorized"}, nil
	}
	if !role.CanManageWorkspaces() {
		return &oapi.IssuesArchiveForbidden{Error: "only managers may archive issues"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.IssuesArchiveNotFound{Error: "issue not found"}, nil
	}
	issue, err := s.IssueRepo.FindByID(id)
	if err != nil {
		return &oapi.IssuesArchiveNotFound{Error: "issue not found"}, nil
	}
	if role != models.RoleRoot {
		ws, err := s.WorkspaceRepo.FindByID(issue.WorkspaceID)
		if err != nil {
			logErr("issues.archive.workspace", err)
			return &oapi.IssuesArchiveInternalServerError{Error: "failed to load workspace"}, nil
		}
		if !ws.IsOwnedBy(uid) {
			return &oapi.IssuesArchiveForbidden{Error: "only the workspace manager may archive issues in this workspace"}, nil
		}
	}
	issue.Archived = req.Archived
	if err := s.IssueRepo.Update(issue); err != nil {
		logErr("issues.archive", err)
		return &oapi.IssuesArchiveInternalServerError{Error: "failed to archive issue"}, nil
	}
	refreshed, err := s.IssueRepo.FindByID(id)
	if err != nil {
		logErr("issues.archive.refetch", err)
		return &oapi.IssuesArchiveInternalServerError{Error: "failed to load updated issue"}, nil
	}
	return &oapi.GetIssueResponse{
		Issue: issueToOAPI(refreshed),
		User:  userToOAPI(&refreshed.Assignee),
	}, nil
}
