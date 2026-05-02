package handlers

import (
	"context"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/oapi"
)

// HandleBearerAuth validates the Authorization: Bearer <jwt> token.
func (s *Server) HandleBearerAuth(ctx context.Context, _ oapi.OperationName, t oapi.BearerAuth) (context.Context, error) {
	if t.Token == "" {
		return ctx, ogenerrors.ErrSecurityRequirementIsNotSatisfied
	}
	out, ok := middleware.Authenticate(ctx, s.Tokens, t.Token)
	if !ok {
		return ctx, ogenerrors.ErrSecurityRequirementIsNotSatisfied
	}
	return out, nil
}

// HandleCookieAuth validates the `session` cookie.
func (s *Server) HandleCookieAuth(ctx context.Context, _ oapi.OperationName, t oapi.CookieAuth) (context.Context, error) {
	if t.APIKey == "" {
		return ctx, ogenerrors.ErrSecurityRequirementIsNotSatisfied
	}
	out, ok := middleware.Authenticate(ctx, s.Tokens, t.APIKey)
	if !ok {
		return ctx, ogenerrors.ErrSecurityRequirementIsNotSatisfied
	}
	return out, nil
}
