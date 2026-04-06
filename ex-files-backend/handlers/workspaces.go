package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/spburtsev/ex-files-backend/gen/auth/v1"
	workspacesv1 "github.com/spburtsev/ex-files-backend/gen/workspaces/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type WorkspaceHandler struct {
	Repo     services.WorkspaceRepository
	UserRepo services.UserRepository
	Audit    services.AuditRepository
}

func workspaceToProto(ws *models.Workspace) *workspacesv1.Workspace {
	return &workspacesv1.Workspace{
		Id:        uint64(ws.ID),
		Name:      ws.Name,
		ManagerId: uint64(ws.ManagerID),
		CreatedAt: timestamppb.New(ws.CreatedAt),
		UpdatedAt: timestamppb.New(ws.UpdatedAt),
	}
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}

	if !role.CanManageWorkspaces() {
		c.JSON(http.StatusForbidden, gin.H{"error": "only managers can create workspaces"})
		return
	}

	var req workspacesv1.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ws := models.Workspace{
		Name:      req.Name,
		ManagerID: userID,
	}
	if err := h.Repo.Create(&ws); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workspace"})
		return
	}

	logAudit(h.Audit, models.AuditActionWorkspaceCreated, userID, uintPtr(ws.ID), "workspace", map[string]any{
		"name": ws.Name,
	})

	protobufResponse(c, http.StatusCreated, &workspacesv1.CreateWorkspaceResponse{
		Workspace: workspaceToProto(&ws),
	})
}

func (h *WorkspaceHandler) List(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}

	page, perPage := parsePagination(c)
	offset := (page - 1) * perPage

	var workspaces []models.Workspace
	var total int64
	var err error

	if role.CanManageWorkspaces() {
		workspaces, total, err = h.Repo.FindByManager(userID, perPage, offset)
	} else {
		workspaces, total, err = h.Repo.FindByMember(userID, perPage, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch workspaces"})
		return
	}

	setPaginationHeaders(c, page, perPage, total)

	pbWorkspaces := make([]*workspacesv1.Workspace, len(workspaces))
	for i := range workspaces {
		pbWorkspaces[i] = workspaceToProto(&workspaces[i])
	}

	protobufResponse(c, http.StatusOK, &workspacesv1.GetWorkspacesResponse{
		Workspaces: pbWorkspaces,
	})
}

func (h *WorkspaceHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	ws, err := h.Repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	manager, err := h.UserRepo.FindByID(ws.ManagerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch manager"})
		return
	}

	members, err := h.Repo.GetMembers(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch members"})
		return
	}

	pbMembers := make([]*authv1.User, len(members))
	for i := range members {
		pbMembers[i] = userToProto(&members[i])
	}

	protobufResponse(c, http.StatusOK, &workspacesv1.GetWorkspaceResponse{
		Workspace: &workspacesv1.WorkspaceDetail{
			Workspace: workspaceToProto(ws),
			Manager:   userToProto(manager),
			Members:   pbMembers,
		},
	})
}

func (h *WorkspaceHandler) AssignableMembers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	users, err := h.Repo.GetAssignableUsers(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch assignable users"})
		return
	}

	pb := make([]*authv1.User, len(users))
	for i := range users {
		pb[i] = userToProto(&users[i])
	}

	protobufResponse(c, http.StatusOK, &workspacesv1.GetAssignableMembersResponse{
		Users: pb,
	})
}

func (h *WorkspaceHandler) Update(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	ws, err := h.Repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	if !ws.IsOwnedBy(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the workspace manager can update it"})
		return
	}

	var req workspacesv1.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ws.Name = req.Name
	if err := h.Repo.Update(ws); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update workspace"})
		return
	}

	logAudit(h.Audit, models.AuditActionWorkspaceUpdated, userID, uintPtr(ws.ID), "workspace", map[string]any{
		"name": ws.Name,
	})

	protobufResponse(c, http.StatusOK, &workspacesv1.UpdateWorkspaceResponse{
		Workspace: workspaceToProto(ws),
	})
}

func (h *WorkspaceHandler) Delete(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	ws, err := h.Repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	if !ws.IsOwnedBy(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the workspace manager can delete it"})
		return
	}

	if err := h.Repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete workspace"})
		return
	}

	logAudit(h.Audit, models.AuditActionWorkspaceDeleted, userID, uintPtr(uint(id)), "workspace", map[string]any{
		"name": ws.Name,
	})

	protobufResponse(c, http.StatusOK, &workspacesv1.DeleteWorkspaceResponse{
		Message: "workspace deleted",
	})
}

func (h *WorkspaceHandler) AddMember(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	ws, err := h.Repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	if !ws.IsOwnedBy(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the workspace manager can add members"})
		return
	}

	var req workspacesv1.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member := models.WorkspaceMember{
		WorkspaceID: uint(id),
		UserID:      uint(req.UserId),
	}
	if err := h.Repo.AddMember(&member); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
		return
	}

	logAudit(h.Audit, models.AuditActionMemberAdded, userID, uintPtr(uint(id)), "workspace", map[string]any{
		"member_user_id": req.UserId,
	})

	protobufResponse(c, http.StatusCreated, &workspacesv1.AddMemberResponse{
		Member: &workspacesv1.WorkspaceMember{
			Id:          uint64(member.ID),
			WorkspaceId: uint64(member.WorkspaceID),
			UserId:      uint64(member.UserID),
			CreatedAt:   timestamppb.New(member.CreatedAt),
		},
	})
}

func (h *WorkspaceHandler) RemoveMember(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	ws, err := h.Repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	if !ws.IsOwnedBy(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the workspace manager can remove members"})
		return
	}

	memberUserID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.Repo.RemoveMember(uint(id), uint(memberUserID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}

	logAudit(h.Audit, models.AuditActionMemberRemoved, userID, uintPtr(uint(id)), "workspace", map[string]any{
		"member_user_id": memberUserID,
	})

	protobufResponse(c, http.StatusOK, &workspacesv1.RemoveMemberResponse{
		Message: "member removed",
	})
}
