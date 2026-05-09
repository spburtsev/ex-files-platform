package handlers

import (
	"context"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

// IssuesListMine implements GET /issues/mine.
//
// Scoped to the authenticated user: returns issues where the caller is
// assignee OR creator AND the issue is unresolved + unarchived. Each row is
// flattened with the parent workspace's id, name, and manager name so the
// dashboard table renders without a follow-up call.
func (s *Server) IssuesListMine(ctx context.Context, params oapi.IssuesListMineParams) (oapi.IssuesListMineRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.IssuesListMineUnauthorized{Error: "unauthorized"}, nil
	}

	page, perPage, offset := resolvePagination(params.Page, params.PerPage)
	search := params.Search.Or("")

	issues, total, err := s.IssueRepo.ListMyCurrentIssues(uid, search, perPage, offset)
	if err != nil {
		logErr("issues.list_mine", err)
		return &oapi.IssuesListMineInternalServerError{Error: "failed to list issues"}, nil
	}

	out := make([]oapi.MyIssueListItem, len(issues))
	for i := range issues {
		out[i] = myIssueListItemToOAPI(&issues[i])
	}

	return &oapi.GetMyIssuesResponseHeaders{
		XPage:       optInt32(page),
		XPerPage:    optInt32(perPage),
		XTotalCount: optInt64(total),
		XTotalPages: optInt32(totalPages(total, perPage)),
		Response:    oapi.GetMyIssuesResponse{Issues: out},
	}, nil
}

func myIssueListItemToOAPI(i *models.Issue) oapi.MyIssueListItem {
	out := oapi.MyIssueListItem{
		ID:                   formatID(i.ID),
		Title:                i.Title,
		WorkspaceId:          formatID(i.WorkspaceID),
		WorkspaceName:        i.Workspace.Name,
		WorkspaceManagerName: i.Workspace.Manager.Name,
		CreatedAt:            i.CreatedAt,
	}
	if i.Deadline != nil {
		out.Deadline = oapi.NewOptNilDateTime(*i.Deadline)
	}
	return out
}
