package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	assignv1 "github.com/spburtsev/ex-files-backend/gen/assignments/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type AssignmentsHandler struct {
	Repo     services.AssignmentRepository
	UserRepo services.UserRepository
}

func assignmentToProto(a *models.Assignment) *assignv1.Assignment {
	pb := &assignv1.Assignment{
		Id:            strconv.FormatUint(uint64(a.ID), 10),
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

func userToAssignProto(u *models.User) *assignv1.User {
	role := assignv1.Role_ROLE_EMPLOYEE
	if u.Role == models.RoleManager || u.Role == models.RoleRoot {
		role = assignv1.Role_ROLE_MANAGER
	}
	return &assignv1.User{
		Id:    strconv.FormatUint(uint64(u.ID), 10),
		Name:  u.Name,
		Email: u.Email,
		Role:  role,
	}
}

func (h *AssignmentsHandler) GetUsers(c *gin.Context) {
	users, err := h.UserRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	pb := make([]*assignv1.User, len(users))
	for i := range users {
		pb[i] = userToAssignProto(&users[i])
	}
	protobufResponse(c, http.StatusOK, &assignv1.GetUsersResponse{Users: pb})
}

func (h *AssignmentsHandler) GetAssignments(c *gin.Context) {
	assignments, err := h.Repo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list assignments"})
		return
	}
	pb := make([]*assignv1.Assignment, len(assignments))
	for i := range assignments {
		pb[i] = assignmentToProto(&assignments[i])
	}
	protobufResponse(c, http.StatusOK, &assignv1.GetAssignmentsResponse{Assignments: pb})
}

func (h *AssignmentsHandler) GetAssignment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment id"})
		return
	}
	a, err := h.Repo.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}
	protobufResponse(c, http.StatusOK, &assignv1.GetAssignmentResponse{
		Assignment: assignmentToProto(a),
		User:       userToAssignProto(&a.Assignee),
	})
}
