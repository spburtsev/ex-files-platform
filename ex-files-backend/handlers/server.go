package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"gorm.io/gorm"

	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/oapi"
	"github.com/spburtsev/ex-files-backend/services"
)

// Server implements the ogen-generated Handler and SecurityHandler interfaces.
// It is the single root handler for the JSON HTTP API.
type Server struct {
	UserRepo      services.UserRepository
	Tokens        services.TokenService
	Hasher        services.Hasher
	Audit         services.AuditRepository
	Email         services.EmailService
	Cache         services.CacheService
	ResetTokens   services.ResetTokenStore
	WorkspaceRepo services.WorkspaceRepository
	IssueRepo     services.IssueRepository
	DocumentRepo  services.DocumentRepository
	CommentRepo   services.CommentRepository
	Storage       services.StorageService
	Hub           *services.SSEHub
	DB            *gorm.DB
}

// errUnauthorized is returned by handlers when the auth context is missing -
// this should not happen in practice because ogen's SecurityHandler runs first.
var errUnauthorized = errors.New("missing authentication context")

func (s *Server) callerID(ctx context.Context) (uint, error) {
	uid, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return 0, errUnauthorized
	}
	return uid, nil
}

func (s *Server) callerIDAndRole(ctx context.Context) (uint, models.Role, error) {
	uid, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return 0, "", errUnauthorized
	}
	role, ok := middleware.RoleFromContext(ctx)
	if !ok {
		return 0, "", errUnauthorized
	}
	return uid, role, nil
}

func parseUintID(s string) (uint, bool) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}

func formatID(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

func roleString(r models.Role) oapi.Role {
	switch r {
	case models.RoleRoot:
		return oapi.RoleRoot
	case models.RoleManager:
		return oapi.RoleManager
	default:
		return oapi.RoleEmployee
	}
}

func userToOAPI(u *models.User) oapi.User {
	out := oapi.User{
		ID:        formatID(u.ID),
		Email:     u.Email,
		Name:      u.Name,
		Role:      roleString(u.Role),
		CreatedAt: u.CreatedAt,
	}
	if u.AvatarURL != "" {
		out.AvatarUrl = oapi.NewOptString(u.AvatarURL)
	}
	return out
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// logErr is a tiny convenience for the ubiquitous "log and return 500" path.
func logErr(op string, err error) {
	slog.Error(op, "error", err)
}
