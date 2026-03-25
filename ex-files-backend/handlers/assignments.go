package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	assignv1 "github.com/spburtsev/ex-files-backend/gen/assignments/v1"
)

type AssignmentsHandler struct{}

func deadline(s string) *timestamppb.Timestamp {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}

var mockUsers = []*assignv1.User{
	{Id: "u1", Name: "Alex Johnson", Email: "a.johnson@acme.org", Role: assignv1.Role_ROLE_EMPLOYEE},
	{Id: "u2", Name: "Maria Chen", Email: "m.chen@acme.org", Role: assignv1.Role_ROLE_EMPLOYEE},
	{Id: "u3", Name: "James Wilson", Email: "j.wilson@acme.org", Role: assignv1.Role_ROLE_EMPLOYEE},
	{Id: "u4", Name: "Sofia Martinez", Email: "s.martinez@acme.org", Role: assignv1.Role_ROLE_MANAGER},
}

var mockAssignments = []*assignv1.Assignment{
	{
		Id: "a1", CreatorId: "u4", AssigneeId: "u1",
		Title:       "Sorting Algorithms Report",
		Description: "Implement and benchmark QuickSort, MergeSort, and HeapSort. Analyse worst-case complexity.",
		Deadline:    deadline("2026-03-15T23:59:00"),
		Resolved:    false, CommentsCount: 4, VersionsCount: 2,
	},
	{
		Id: "a2", CreatorId: "u4", AssigneeId: "u1",
		Title:       "Binary Search Trees",
		Description: "Build a BST with insert, delete, and balanced-rotation operations.",
		Resolved:    true, CommentsCount: 5, VersionsCount: 3,
	},
	{
		Id: "a3", CreatorId: "u4", AssigneeId: "u2",
		Title:       "Arrays & Linked Lists",
		Description: "Implement a doubly-linked list and compare performance with dynamic arrays.",
		Resolved:    true, CommentsCount: 3, VersionsCount: 2,
	},
	{
		Id: "a5", CreatorId: "u4", AssigneeId: "u2",
		Title:       "Shell Implementation",
		Description: "Implement a Unix-like shell supporting pipes, redirection, and job control.",
		Deadline:    deadline("2026-03-22T23:59:00"),
		Resolved:    false, CommentsCount: 6, VersionsCount: 4,
	},
	{
		Id: "a6", CreatorId: "u4", AssigneeId: "u3",
		Title:       "Memory Allocator",
		Description: "Build a user-space memory allocator with first-fit and best-fit strategies.",
		Deadline:    deadline("2026-03-20T23:59:00"),
		Resolved:    false, CommentsCount: 2, VersionsCount: 1,
	},
	{
		Id: "a8", CreatorId: "u4", AssigneeId: "u3",
		Title:       "Eigenvalue Analysis",
		Description: "Solve eigenvalue and eigenvector problems for given matrices.",
		Deadline:    deadline("2026-03-16T23:59:00"),
		Resolved:    true, CommentsCount: 1, VersionsCount: 1,
	},
	{
		Id: "a16", CreatorId: "u4", AssigneeId: "u1",
		Title:       "Design Document v1",
		Description: "Create UML class, sequence, and component diagrams for the system design.",
		Deadline:    deadline("2026-03-18T23:59:00"),
		Resolved:    false, CommentsCount: 2, VersionsCount: 1,
	},
	{
		Id: "a17", CreatorId: "u4", AssigneeId: "u2",
		Title:       "Network Protocol Analysis",
		Description: "Capture and analyse TCP/IP traffic. Document findings.",
		Resolved:    true, CommentsCount: 0, VersionsCount: 2,
	},
}

func (h *AssignmentsHandler) GetUsers(c *gin.Context) {
	protobufResponse(c, http.StatusOK, &assignv1.GetUsersResponse{Users: mockUsers})
}

func (h *AssignmentsHandler) GetAssignments(c *gin.Context) {
	protobufResponse(c, http.StatusOK, &assignv1.GetAssignmentsResponse{Assignments: mockAssignments})
}

func (h *AssignmentsHandler) GetAssignment(c *gin.Context) {
	id := c.Param("id")
	var found *assignv1.Assignment
	for _, a := range mockAssignments {
		if a.Id == id {
			found = a
			break
		}
	}
	if found == nil {
		found = mockAssignments[0]
	}
	var user *assignv1.User
	for _, u := range mockUsers {
		if u.Id == found.AssigneeId {
			user = u
			break
		}
	}
	protobufResponse(c, http.StatusOK, &assignv1.GetAssignmentResponse{Assignment: found, User: user})
}
