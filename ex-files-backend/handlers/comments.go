package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/timestamppb"

	docsv1 "github.com/spburtsev/ex-files-backend/gen/documents/v1"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type CommentHandler struct {
	Repo  services.CommentRepository
	Audit services.AuditRepository
	Hub   *services.SSEHub
}

func commentToProto(c *models.Comment) *docsv1.Comment {
	return &docsv1.Comment{
		Id:         uint64(c.ID),
		DocumentId: uint64(c.DocumentID),
		AuthorId:   uint64(c.AuthorID),
		AuthorName: c.Author.Name,
		Body:       c.Body,
		CreatedAt:  timestamppb.New(c.CreatedAt),
	}
}

// Create adds a comment to a document.
// @Summary      Create comment
// @Tags         comments
// @Accept       json
// @Produce      application/x-protobuf
// @Param        id    path      int                       true  "Document ID"
// @Param        body  body      swagCreateCommentRequest  true  "Comment payload"
// @Success      201   {object}  swagCreateCommentResponse "Protobuf: documents.v1.CreateCommentResponse"
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/comments [post]
func (h *CommentHandler) Create(c *gin.Context) {
	userID, ok := mustGetUserID(c)
	if !ok {
		return
	}

	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var body struct {
		Body string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body is required"})
		return
	}

	comment := models.Comment{
		DocumentID: uint(docID),
		AuthorID:   userID,
		Body:       body.Body,
	}
	if err := h.Repo.Create(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create comment"})
		return
	}

	logAudit(h.Audit, models.AuditActionCommentAdded, userID, uintPtr(uint(docID)), "document", map[string]any{
		"comment_id":  comment.ID,
		"document_id": docID,
	})

	created, err := h.Repo.FindByID(comment.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reload comment"})
		return
	}

	h.Hub.Broadcast(services.SSEEvent{
		Type:       "comment.added",
		DocumentID: uint(docID),
	})

	protobufResponse(c, http.StatusCreated, &docsv1.CreateCommentResponse{
		Comment: commentToProto(created),
	})
}

// List returns all comments for a document.
// @Summary      List comments
// @Tags         comments
// @Produce      application/x-protobuf
// @Param        id   path      int  true  "Document ID"
// @Success      200  {object}  swagListCommentsResponse  "Protobuf: documents.v1.ListCommentsResponse"
// @Security     BearerAuth || CookieAuth
// @Router       /documents/{id}/comments [get]
func (h *CommentHandler) List(c *gin.Context) {
	docID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	comments, err := h.Repo.ListByDocument(uint(docID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}

	pb := make([]*docsv1.Comment, len(comments))
	for i := range comments {
		pb[i] = commentToProto(&comments[i])
	}

	protobufResponse(c, http.StatusOK, &docsv1.ListCommentsResponse{
		Comments: pb,
	})
}
