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

// Create creates a new workspace. Only managers and root users.
// @Summary      Create workspace
// @Tags         workspaces
// @Accept       json
// @Produce      application/x-protobuf
// @Param        body  body      swagCreateWorkspaceRequest   true  "Workspace payload"
// @Success      201   {object}  swagCreateWorkspaceResponse  "Protobuf: workspaces.v1.CreateWorkspaceResponse"
// @Failure      400   {object}  swagErrorResponse
// @Failure      403   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces [post]
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

// List returns workspaces for the current user.
// @Summary      List workspaces
// @Tags         workspaces
// @Produce      application/x-protobuf
// @Param        page      query  int  false  "Page number"      default(1)
// @Param        per_page  query  int  false  "Items per page"   default(20)
// @Success      200  {object}  swagGetWorkspacesResponse  "Protobuf: workspaces.v1.GetWorkspacesResponse"
// @Header       200  {int}     X-Total-Count
// @Header       200  {int}     X-Total-Pages
// @Header       200  {int}     X-Page
// @Header       200  {int}     X-Per-Page
// @Failure      401  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces [get]
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

// Get returns a workspace with its manager and members.
// @Summary      Get workspace
// @Tags         workspaces
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Workspace ID"
// @Success      200  {object}  swagGetWorkspaceResponse  "Protobuf: workspaces.v1.GetWorkspaceResponse"
// @Failure      404  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id} [get]
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

// AssignableMembers returns users that can be added to the workspace.
// @Summary      Assignable members
// @Tags         workspaces
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Workspace ID"
// @Success      200  {object}  swagAssignableMembersResponse  "Protobuf: workspaces.v1.GetAssignableMembersResponse"
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id}/assignable-members [get]
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

// Update renames a workspace. Only the workspace manager.
// @Summary      Update workspace
// @Tags         workspaces
// @Accept       json
// @Produce      application/x-protobuf
// @Param        id    path      int                         true  "Workspace ID"
// @Param        body  body      swagUpdateWorkspaceRequest  true  "Update payload"
// @Success      200   {object}  swagUpdateWorkspaceResponse "Protobuf: workspaces.v1.UpdateWorkspaceResponse"
// @Failure      403   {object}  swagErrorResponse
// @Failure      404   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id} [put]
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

// Delete removes a workspace. Only the workspace manager.
// @Summary      Delete workspace
// @Tags         workspaces
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Workspace ID"
// @Success      200  {object}  swagMessageResponse  "Protobuf: workspaces.v1.DeleteWorkspaceResponse"
// @Failure      403  {object}  swagErrorResponse
// @Failure      404  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id} [delete]
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

// AddMember adds a user to a workspace. Only the workspace manager.
// @Summary      Add member
// @Tags         workspaces
// @Accept       json
// @Produce      application/x-protobuf
// @Param        id    path      int                   true  "Workspace ID"
// @Param        body  body      swagAddMemberRequest  true  "Member payload"
// @Success      201   {object}  swagAddMemberResponse "Protobuf: workspaces.v1.AddMemberResponse"
// @Failure      403   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id}/members [post]
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

// RemoveMember removes a user from a workspace. Only the workspace manager.
// @Summary      Remove member
// @Tags         workspaces
// @Produce      application/x-protobuf
// @Param        id      path      int  true  "Workspace ID"
// @Param        userId  path      int  true  "User ID"
// @Success      200     {object}  swagMessageResponse  "Protobuf: workspaces.v1.RemoveMemberResponse"
// @Failure      403     {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id}/members/{userId} [delete]
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
