package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	issuesv1 "github.com/spburtsev/ex-files-backend/gen/issues/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type IssuesHandler struct {
	Repo     services.IssueRepository
	UserRepo services.UserRepository
	Audit    services.AuditRepository
}

func issueToProto(a *models.Issue) *issuesv1.Issue {
	pb := &issuesv1.Issue{
		Id:            strconv.FormatUint(uint64(a.ID), 10),
		WorkspaceId:   strconv.FormatUint(uint64(a.WorkspaceID), 10),
		CreatorId:     strconv.FormatUint(uint64(a.CreatorID), 10),
		AssigneeId:    strconv.FormatUint(uint64(a.AssigneeID), 10),
		Title:         a.Title,
		Description:   a.Description,
		Resolved:      a.Resolved,
		CommentsCount: int32(a.CommentsCount),
		VersionsCount: int32(a.VersionsCount),
	}
	if a.Deadline != nil {
		pb.Deadline = timestamppb.New(*a.Deadline)
	}
	return pb
}

func userToIssueProto(u *models.User) *issuesv1.User {
	role := issuesv1.Role_ROLE_EMPLOYEE
	if u.Role == models.RoleManager || u.Role == models.RoleRoot {
		role = issuesv1.Role_ROLE_MANAGER
	}
	return &issuesv1.User{
		Id:    strconv.FormatUint(uint64(u.ID), 10),
		Name:  u.Name,
		Email: u.Email,
		Role:  role,
	}
}

func (h *IssuesHandler) GetUsers(c *gin.Context) {
	users, err := h.UserRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	pb := make([]*issuesv1.User, len(users))
	for i := range users {
		pb[i] = userToIssueProto(&users[i])
	}
	protobufResponse(c, http.StatusOK, &issuesv1.GetUsersResponse{Users: pb})
}

// ListByWorkspace returns all issues for a workspace.
// @Summary      List issues by workspace
// @Tags         issues
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Workspace ID"
// @Success      200  {object}  swagGetIssuesResponse  "Protobuf: issues.v1.GetIssuesResponse"
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id}/issues [get]
func (h *IssuesHandler) ListByWorkspace(c *gin.Context) {
	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}
	issues, err := h.Repo.ListByWorkspace(uint(wsID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list issues"})
		return
	}
	pb := make([]*issuesv1.Issue, len(issues))
	for i := range issues {
		pb[i] = issueToProto(&issues[i])
	}
	protobufResponse(c, http.StatusOK, &issuesv1.GetIssuesResponse{Issues: pb})
}

// Get returns a single issue with its assignee.
// @Summary      Get issue
// @Tags         issues
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Issue ID"
// @Success      200  {object}  swagGetIssueResponse  "Protobuf: issues.v1.GetIssueResponse"
// @Failure      404  {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /issues/{id} [get]
func (h *IssuesHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid issue id"})
		return
	}
	issue, err := h.Repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "issue not found"})
		return
	}
	protobufResponse(c, http.StatusOK, &issuesv1.GetIssueResponse{
		Issue: issueToProto(issue),
		User:  userToIssueProto(&issue.Assignee),
	})
}

// Create creates a new issue in a workspace. Only managers and root users.
// @Summary      Create issue
// @Tags         issues
// @Accept       json
// @Produce      application/x-protobuf
// @Param        id    path      int                     true  "Workspace ID"
// @Param        body  body      swagCreateIssueRequest  true  "Issue payload"
// @Success      201   {object}  swagCreateIssueResponse "Protobuf: issues.v1.CreateIssueResponse"
// @Failure      403   {object}  swagErrorResponse
// @Security     BearerAuth || CookieAuth
// @Router       /workspaces/{id}/issues [post]
func (h *IssuesHandler) Create(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}
	role, ok := mustGetRole(c)
	if !ok {
		return
	}

	if !role.CanManageWorkspaces() {
		c.JSON(http.StatusForbidden, gin.H{"error": "only managers may create issues"})
		return
	}

	wsID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace id"})
		return
	}

	var body struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		AssigneeID  uint   `json:"assignee_id" binding:"required"`
		Deadline    string `json:"deadline"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and assignee_id are required"})
		return
	}

	issue := models.Issue{
		WorkspaceID: uint(wsID),
		CreatorID:   userID,
		AssigneeID:  body.AssigneeID,
		Title:       body.Title,
		Description: body.Description,
	}

	if body.Deadline != "" {
		t, err := parseTime(body.Deadline)
		if err == nil {
			issue.Deadline = &t
		}
	}

	if err := h.Repo.Create(&issue); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create issue"})
		return
	}

	protobufResponse(c, http.StatusCreated, &issuesv1.CreateIssueResponse{
		Issue: issueToProto(&issue),
	})
}
