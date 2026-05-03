package handlers

import (
	"context"
	"encoding/json"

	"gorm.io/datatypes"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

// jsonNumberToFloat handles values from datatypes.JSONMap.Scan, which uses
// json.Decoder.UseNumber() — numeric fields come back as json.Number (string),
// not float64.
func jsonNumberToFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func commentToOAPI(c *models.Comment) oapi.Comment {
	meta := oapi.CommentMetadata{}
	if c.Metadata != nil {
		if f, ok := jsonNumberToFloat(c.Metadata["page"]); ok {
			meta.Page = int(f)
		}
		if f, ok := jsonNumberToFloat(c.Metadata["x"]); ok {
			meta.X = f
		}
		if f, ok := jsonNumberToFloat(c.Metadata["y"]); ok {
			meta.Y = f
		}
	}
	return oapi.Comment{
		ID:         formatID(c.ID),
		DocumentId: formatID(c.DocumentID),
		AuthorId:   formatID(c.AuthorID),
		AuthorName: c.Author.Name,
		Body:       c.Body,
		Metadata:   meta,
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
		Metadata: datatypes.JSONMap{
			"page": req.Metadata.Page,
			"x":    req.Metadata.X,
			"y":    req.Metadata.Y,
		},
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

// CommentsDelete implements DELETE /documents/{id}/comments/{commentId}.
func (s *Server) CommentsDelete(ctx context.Context, params oapi.CommentsDeleteParams) (oapi.CommentsDeleteRes, error) {
	uid, err := s.callerID(ctx)
	if err != nil {
		return &oapi.CommentsDeleteUnauthorized{Error: "unauthorized"}, nil
	}
	commentID, ok := parseUintID(params.CommentId)
	if !ok {
		return &oapi.CommentsDeleteNotFound{Error: "comment not found"}, nil
	}
	c, err := s.CommentRepo.FindByID(commentID)
	if err != nil {
		return &oapi.CommentsDeleteNotFound{Error: "comment not found"}, nil
	}
	if c.AuthorID != uid {
		return &oapi.CommentsDeleteForbidden{Error: "only the author can delete this comment"}, nil
	}
	if err := s.CommentRepo.Delete(commentID); err != nil {
		logErr("comments.delete", err)
		return &oapi.CommentsDeleteInternalServerError{Error: "failed to delete comment"}, nil
	}
	logAudit(s.Audit, models.AuditActionCommentDeleted, uid, uintPtr(c.ID), "comment", map[string]any{
		"document_id": c.DocumentID,
	})
	if s.Hub != nil {
		s.Hub.Broadcast(services.SSEEvent{
			Type:       "comment.deleted",
			DocumentID: c.DocumentID,
			Payload:    map[string]any{"id": c.ID},
		})
	}
	return &oapi.CommentsDeleteNoContent{}, nil
}
