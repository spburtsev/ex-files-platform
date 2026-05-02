package handlers

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/oapi"
)

// VerifyHash implements GET /verify.
func (s *Server) VerifyHash(ctx context.Context, params oapi.VerifyHashParams) (oapi.VerifyHashRes, error) {
	if params.Hash == "" {
		return &oapi.VerifyHashBadRequest{Error: "hash query parameter is required"}, nil
	}

	doc, err := s.DocumentRepo.FindByHash(params.Hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &oapi.VerifyResponse{Verified: false}, nil
		}
		logErr("verify.lookup", err)
		return &oapi.VerifyHashInternalServerError{Error: "failed to look up document"}, nil
	}

	resp := oapi.VerifyResponse{
		Verified:     true,
		DocumentName: oapi.NewOptString(doc.Name),
		Status:       oapi.NewOptDocumentStatus(oapi.DocumentStatus(doc.Status)),
		NotarizedAt:  oapi.NewOptDateTime(doc.CreatedAt),
		Hash:         oapi.NewOptString(doc.Hash),
	}
	return &resp, nil
}
