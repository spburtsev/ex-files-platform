package handlers

import (
	"github.com/spburtsev/ex-files-backend/models"
)

// canViewWorkspace reports whether the caller may see a workspace and the
// issues, documents and comments inside it: root, the workspace manager, or
// an active member.
func (s *Server) canViewWorkspace(ws *models.Workspace, uid uint, role models.Role) (bool, error) {
	if role == models.RoleRoot || ws.IsOwnedBy(uid) {
		return true, nil
	}
	return s.WorkspaceRepo.IsMember(ws.ID, uid)
}

// canViewIssue extends workspace access with the issue's direct participants
// (creator, assignee, reviewers), who are not necessarily workspace members.
// The issue must have Reviewers preloaded (IssueRepo.FindByID does).
func (s *Server) canViewIssue(issue *models.Issue, uid uint, role models.Role) (bool, error) {
	if issue.CreatorID == uid || issue.AssigneeID == uid {
		return true, nil
	}
	for i := range issue.Reviewers {
		if issue.Reviewers[i].UserID == uid {
			return true, nil
		}
	}
	ws, err := s.WorkspaceRepo.FindByID(issue.WorkspaceID)
	if err != nil {
		return false, err
	}
	return s.canViewWorkspace(ws, uid, role)
}

// canViewDocument resolves the document's issue and applies issue-level
// access. The returned issue is non-nil whenever it had to be loaded so
// callers can reuse it; it is nil when the uploader short-circuit applied.
func (s *Server) canViewDocument(doc *models.Document, uid uint, role models.Role) (bool, *models.Issue, error) {
	if doc.UploaderID == uid {
		return true, nil, nil
	}
	issue, err := s.IssueRepo.FindByID(doc.IssueID)
	if err != nil {
		return false, nil, err
	}
	ok, err := s.canViewIssue(issue, uid, role)
	return ok, issue, err
}
