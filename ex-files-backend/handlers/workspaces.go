package handlers

import (
	"context"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
)

func workspaceToOAPI(ws *models.Workspace) oapi.Workspace {
	status := oapi.WorkspaceStatus(ws.Status)
	if status == "" {
		status = oapi.WorkspaceStatusActive
	}
	return oapi.Workspace{
		ID:          formatID(ws.ID),
		Name:        ws.Name,
		Status:      status,
		ManagerId:   formatID(ws.ManagerID),
		ManagerName: ws.Manager.Name,
		CreatedAt:   ws.CreatedAt,
		UpdatedAt:   ws.UpdatedAt,
	}
}

// WorkspacesCreate implements POST /workspaces.
func (s *Server) WorkspacesCreate(ctx context.Context, req *oapi.CreateWorkspaceRequest) (oapi.WorkspacesCreateRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.WorkspacesCreateUnauthorized{Error: "unauthorized"}, nil
	}
	if !role.CanManageWorkspaces() {
		return &oapi.WorkspacesCreateForbidden{Error: "only managers can create workspaces"}, nil
	}

	ws := models.Workspace{Name: req.Name, Status: models.WorkspaceStatusActive, ManagerID: uid}
	if err := s.WorkspaceRepo.Create(&ws); err != nil {
		logErr("workspaces.create", err)
		return &oapi.WorkspacesCreateInternalServerError{Error: "failed to create workspace"}, nil
	}
	if mgr, err := s.UserRepo.FindByID(uid); err == nil && mgr != nil {
		ws.Manager = *mgr
	}

	logAudit(s.Audit, models.AuditActionWorkspaceCreated, uid, uintPtr(ws.ID), "workspace", map[string]any{
		"name": ws.Name,
	})

	return &oapi.CreateWorkspaceResponse{Workspace: workspaceToOAPI(&ws)}, nil
}

// WorkspacesList implements GET /workspaces.
func (s *Server) WorkspacesList(ctx context.Context, params oapi.WorkspacesListParams) (oapi.WorkspacesListRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.WorkspacesListUnauthorized{Error: "unauthorized"}, nil
	}

	page, perPage, offset := resolvePagination(params.Page, params.PerPage)
	search := params.Search.Or("")
	statusParam := params.Status.Or(oapi.WorkspacesListStatusActive)
	var status models.WorkspaceStatus
	if statusParam != oapi.WorkspacesListStatusAll {
		status = models.WorkspaceStatus(statusParam)
	}

	var (
		ws    []models.Workspace
		total int64
	)
	if role.CanManageWorkspaces() {
		ws, total, err = s.WorkspaceRepo.FindByManager(uid, search, status, perPage, offset)
	} else {
		ws, total, err = s.WorkspaceRepo.FindByMember(uid, search, status, perPage, offset)
	}
	if err != nil {
		logErr("workspaces.list", err)
		return &oapi.WorkspacesListInternalServerError{Error: "failed to fetch workspaces"}, nil
	}

	out := make([]oapi.Workspace, len(ws))
	for i := range ws {
		out[i] = workspaceToOAPI(&ws[i])
	}

	return &oapi.GetWorkspacesResponseHeaders{
		XPage:       optInt32(page),
		XPerPage:    optInt32(perPage),
		XTotalCount: optInt64(total),
		XTotalPages: optInt32(totalPages(total, perPage)),
		Response:    oapi.GetWorkspacesResponse{Workspaces: out},
	}, nil
}

// WorkspacesGet implements GET /workspaces/{id}.
func (s *Server) WorkspacesGet(ctx context.Context, params oapi.WorkspacesGetParams) (oapi.WorkspacesGetRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.WorkspacesGetUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.WorkspacesGetNotFound{Error: "workspace not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(id)
	if err != nil {
		return &oapi.WorkspacesGetNotFound{Error: "workspace not found"}, nil
	}
	manager, err := s.UserRepo.FindByID(ws.ManagerID)
	if err != nil {
		logErr("workspaces.get.manager", err)
		return &oapi.WorkspacesGetInternalServerError{Error: "failed to fetch manager"}, nil
	}
	ws.Manager = *manager
	members, err := s.WorkspaceRepo.GetMembers(id)
	if err != nil {
		logErr("workspaces.get.members", err)
		return &oapi.WorkspacesGetInternalServerError{Error: "failed to fetch members"}, nil
	}
	pbMembers := make([]oapi.User, len(members))
	for i := range members {
		pbMembers[i] = userToOAPI(&members[i])
	}
	return &oapi.GetWorkspaceResponse{
		Workspace: oapi.WorkspaceDetail{
			Workspace: workspaceToOAPI(ws),
			Manager:   userToOAPI(manager),
			Members:   pbMembers,
		},
	}, nil
}

// WorkspacesAssignableMembers implements GET /workspaces/{id}/assignable-members.
func (s *Server) WorkspacesAssignableMembers(ctx context.Context, params oapi.WorkspacesAssignableMembersParams) (oapi.WorkspacesAssignableMembersRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.WorkspacesAssignableMembersUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.WorkspacesAssignableMembersInternalServerError{Error: "invalid workspace id"}, nil
	}
	users, err := s.WorkspaceRepo.GetAssignableUsers(id)
	if err != nil {
		logErr("workspaces.assignable", err)
		return &oapi.WorkspacesAssignableMembersInternalServerError{Error: "failed to fetch assignable users"}, nil
	}
	out := make([]oapi.User, len(users))
	for i := range users {
		out[i] = userToOAPI(&users[i])
	}
	return &oapi.AssignableMembersResponse{Users: out}, nil
}

// WorkspacesUpdate implements PUT /workspaces/{id}.
func (s *Server) WorkspacesUpdate(ctx context.Context, req *oapi.UpdateWorkspaceRequest, params oapi.WorkspacesUpdateParams) (oapi.WorkspacesUpdateRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.WorkspacesUpdateUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.WorkspacesUpdateNotFound{Error: "workspace not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(id)
	if err != nil {
		return &oapi.WorkspacesUpdateNotFound{Error: "workspace not found"}, nil
	}
	if !ws.IsOwnedBy(uid) {
		return &oapi.WorkspacesUpdateForbidden{Error: "only the workspace manager can update it"}, nil
	}
	ws.Name = req.Name
	if err := s.WorkspaceRepo.Update(ws); err != nil {
		logErr("workspaces.update", err)
		return &oapi.WorkspacesUpdateInternalServerError{Error: "failed to update workspace"}, nil
	}
	if mgr, err := s.UserRepo.FindByID(ws.ManagerID); err == nil && mgr != nil {
		ws.Manager = *mgr
	}
	logAudit(s.Audit, models.AuditActionWorkspaceUpdated, uid, uintPtr(ws.ID), "workspace", map[string]any{
		"name": ws.Name,
	})
	return &oapi.UpdateWorkspaceResponse{Workspace: workspaceToOAPI(ws)}, nil
}

// WorkspacesDelete implements DELETE /workspaces/{id}.
func (s *Server) WorkspacesDelete(ctx context.Context, params oapi.WorkspacesDeleteParams) (oapi.WorkspacesDeleteRes, error) {
	uid, role, err := s.callerIDAndRole(ctx)
	if err != nil {
		return &oapi.WorkspacesDeleteUnauthorized{Error: "unauthorized"}, nil
	}
	if role != models.RoleRoot {
		return &oapi.WorkspacesDeleteForbidden{Error: "only root may delete workspaces"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.WorkspacesDeleteNotFound{Error: "workspace not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(id)
	if err != nil {
		return &oapi.WorkspacesDeleteNotFound{Error: "workspace not found"}, nil
	}
	if err := s.WorkspaceRepo.Delete(id); err != nil {
		logErr("workspaces.delete", err)
		return &oapi.WorkspacesDeleteInternalServerError{Error: "failed to delete workspace"}, nil
	}
	logAudit(s.Audit, models.AuditActionWorkspaceDeleted, uid, uintPtr(id), "workspace", map[string]any{
		"name": ws.Name,
	})
	return &oapi.MessageResponse{Message: "workspace deleted"}, nil
}

// WorkspacesArchive implements PUT /workspaces/{id}/archive.
func (s *Server) WorkspacesArchive(ctx context.Context, params oapi.WorkspacesArchiveParams) (oapi.WorkspacesArchiveRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.WorkspacesArchiveUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.WorkspacesArchiveNotFound{Error: "workspace not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(id)
	if err != nil {
		return &oapi.WorkspacesArchiveNotFound{Error: "workspace not found"}, nil
	}
	if !ws.IsOwnedBy(uid) {
		return &oapi.WorkspacesArchiveForbidden{Error: "only the workspace manager can archive it"}, nil
	}
	ws.Status = models.WorkspaceStatusArchived
	if err := s.WorkspaceRepo.Update(ws); err != nil {
		logErr("workspaces.archive", err)
		return &oapi.WorkspacesArchiveInternalServerError{Error: "failed to archive workspace"}, nil
	}
	logAudit(s.Audit, models.AuditActionWorkspaceUpdated, uid, uintPtr(id), "workspace", map[string]any{
		"name": ws.Name, "status": "archived",
	})
	return &oapi.MessageResponse{Message: "workspace archived"}, nil
}

// WorkspacesAddMember implements POST /workspaces/{id}/members.
func (s *Server) WorkspacesAddMember(ctx context.Context, req *oapi.AddMemberRequest, params oapi.WorkspacesAddMemberParams) (oapi.WorkspacesAddMemberRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.WorkspacesAddMemberUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.WorkspacesAddMemberNotFound{Error: "workspace not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(id)
	if err != nil {
		return &oapi.WorkspacesAddMemberNotFound{Error: "workspace not found"}, nil
	}
	if !ws.IsOwnedBy(uid) {
		return &oapi.WorkspacesAddMemberForbidden{Error: "only the workspace manager can add members"}, nil
	}
	memberID, ok := parseUintID(req.UserId)
	if !ok {
		return &oapi.WorkspacesAddMemberBadRequest{Error: "invalid userId"}, nil
	}
	member := models.WorkspaceMember{
		WorkspaceID: id,
		UserID:      memberID,
	}
	if err := s.WorkspaceRepo.AddMember(&member); err != nil {
		logErr("workspaces.add_member", err)
		return &oapi.WorkspacesAddMemberInternalServerError{Error: "failed to add member"}, nil
	}
	logAudit(s.Audit, models.AuditActionMemberAdded, uid, uintPtr(id), "workspace", map[string]any{
		"member_user_id": memberID,
	})
	return &oapi.AddMemberResponse{
		Member: oapi.WorkspaceMember{
			ID:          formatID(member.ID),
			WorkspaceId: formatID(member.WorkspaceID),
			UserId:      formatID(member.UserID),
			CreatedAt:   member.CreatedAt,
		},
	}, nil
}

// WorkspacesRemoveMember implements DELETE /workspaces/{id}/members/{userId}.
func (s *Server) WorkspacesRemoveMember(ctx context.Context, params oapi.WorkspacesRemoveMemberParams) (oapi.WorkspacesRemoveMemberRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.WorkspacesRemoveMemberUnauthorized{Error: "unauthorized"}, nil
	}
	id, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.WorkspacesRemoveMemberNotFound{Error: "workspace not found"}, nil
	}
	ws, err := s.WorkspaceRepo.FindByID(id)
	if err != nil {
		return &oapi.WorkspacesRemoveMemberNotFound{Error: "workspace not found"}, nil
	}
	if !ws.IsOwnedBy(uid) {
		return &oapi.WorkspacesRemoveMemberForbidden{Error: "only the workspace manager can remove members"}, nil
	}
	memberID, ok := parseUintID(params.UserId)
	if !ok {
		return &oapi.WorkspacesRemoveMemberNotFound{Error: "member not found"}, nil
	}
	if err := s.WorkspaceRepo.RemoveMember(id, memberID); err != nil {
		logErr("workspaces.remove_member", err)
		return &oapi.WorkspacesRemoveMemberInternalServerError{Error: "failed to remove member"}, nil
	}
	logAudit(s.Audit, models.AuditActionMemberRemoved, uid, uintPtr(id), "workspace", map[string]any{
		"member_user_id": memberID,
	})
	return &oapi.MessageResponse{Message: "member removed"}, nil
}
