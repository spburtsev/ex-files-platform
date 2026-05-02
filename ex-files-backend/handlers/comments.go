package handlers

import (
	"context"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

func commentToOAPI(c *models.Comment) oapi.Comment {
	return oapi.Comment{
		ID:         formatID(c.ID),
		DocumentId: formatID(c.DocumentID),
		AuthorId:   formatID(c.AuthorID),
		AuthorName: c.Author.Name,
		Body:       c.Body,
		CreatedAt:  c.CreatedAt,
	}
}

// CommentsCreate implements POST /documents/{id}/comments.
func (s *Server) CommentsCreate(ctx context.Context, req *oapi.CreateCommentRequest, params oapi.CommentsCreateParams) (oapi.CommentsCreateRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.CommentsCreateUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.CommentsCreateBadRequest{Error: "invalid document id"}, nil
	}
	c := models.Comment{
		DocumentID: docID,
		AuthorID:   uid,
		Body:       req.Body,
	}
	if err := s.CommentRepo.Create(&c); err != nil {
		logErr("comments.create", err)
		return &oapi.CommentsCreateInternalServerError{Error: "failed to create comment"}, nil
	}
	logAudit(s.Audit, models.AuditActionCommentAdded, uid, uintPtr(docID), "document", map[string]any{
		"comment_id":  c.ID,
		"document_id": docID,
	})
	created, err := s.CommentRepo.FindByID(c.ID)
	if err != nil {
		logErr("comments.create.reload", err)
		return &oapi.CommentsCreateInternalServerError{Error: "failed to reload comment"}, nil
	}
	if s.Hub != nil {
		s.Hub.Broadcast(services.SSEEvent{
			Type:       "comment.added",
			DocumentID: docID,
		})
	}
	return &oapi.CreateCommentResponse{Comment: commentToOAPI(created)}, nil
}

// CommentsList implements GET /documents/{id}/comments.
func (s *Server) CommentsList(ctx context.Context, params oapi.CommentsListParams) (oapi.CommentsListRes, error) {
	if _, err := s.callerID(ctx); err != nil {
		return &oapi.CommentsListUnauthorized{Error: "unauthorized"}, nil
	}
	docID, ok := parseUintID(params.ID)
	if !ok {
		return &oapi.CommentsListInternalServerError{Error: "invalid document id"}, nil
	}
	comments, err := s.CommentRepo.ListByDocument(docID)
	if err != nil {
		logErr("comments.list", err)
		return &oapi.CommentsListInternalServerError{Error: "failed to fetch comments"}, nil
	}
	out := make([]oapi.Comment, len(comments))
	for i := range comments {
		out[i] = commentToOAPI(&comments[i])
	}
	return &oapi.ListCommentsResponse{Comments: out}, nil
}
