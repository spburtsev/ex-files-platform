package handlers

import (
	"context"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func issueToOAPI(i *models.Issue) oapi.Issue {
	out := oapi.Issue{
		ID:            formatID(i.ID),
		WorkspaceId:   formatID(i.WorkspaceID),
		CreatorId:     formatID(i.CreatorID),
		AssigneeId:    formatID(i.AssigneeID),
		Title:         i.Title,
		Description:   i.Description,
		Resolved:      i.Resolved,
		Archived:      oapi.NewOptBool(i.Archived),
		CommentsCount: int32(i.CommentsCount),
		VersionsCount: int32(i.VersionsCount),
	}
	if i.Deadline != nil {
		out.Deadline = oapi.NewOptNilDateTime(*i.Deadline)
	}
	return out
}

// IssuesGet implements GET /issues/{id}.
func (s *Server) IssuesGet(ctx context.Context, params oapi.IssuesGetParams) (oapi.IssuesGetRes, error) {
	if _, err := s.callerID(ctx); err != nil {
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
	return &oapi.GetIssueResponse{
		Issue: issueToOAPI(issue),
		User:  userToOAPI(&issue.Assignee),
	}, nil
}

// IssuesListByWorkspace implements GET /workspaces/{id}/issues.
func (s *Server) IssuesListByWorkspace(ctx context.Context, params oapi.IssuesListByWorkspaceParams) (oapi.IssuesListByWorkspaceRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.IssuesListByWorkspaceUnauthorized{Error: "unauthorized"}, nil
	}
	wsID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.IssuesListByWorkspaceInternalServerError{Error: "invalid workspace id"}, nil
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
	if !role.CanManageWorkspaces() && issue.CreatorID != uid {
		return &oapi.IssuesUpdateAssigneeForbidden{Error: "only managers or the issue creator may change the assignee"}, nil
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
	assigneeID, ok := parseUintID(req.AssigneeId)
	if !ok {
		return &oapi.IssuesCreateBadRequest{Error: "invalid assigneeId"}, nil
	}

	issue := models.Issue{
		WorkspaceID: wsID,
		CreatorID:   uid,
		AssigneeID:  assigneeID,
		Title:       req.Title,
		Description: req.Description.Or(""),
	}
	if d, ok := req.Deadline.Get(); ok {
		issue.Deadline = &d
	}
	if err := s.IssueRepo.Create(&issue); err != nil {
		logErr("issues.create", err)
		return &oapi.IssuesCreateInternalServerError{Error: "failed to create issue"}, nil
	}
	return &oapi.CreateIssueResponse{Issue: issueToOAPI(&issue)}, nil
}

// IssuesArchive implements PUT /issues/{id}/archive.
func (s *Server) IssuesArchive(ctx context.Context, req *oapi.ArchiveIssueRequest, params oapi.IssuesArchiveParams) (oapi.IssuesArchiveRes, error) {
	_, role, err := s.callerIDAndRole(ctx)
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
