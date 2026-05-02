package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/spburtsev/ex-files-backend/models"
	"github.com/spburtsev/ex-files-backend/services"
)

type ctxKey string

const (
	userIDKey ctxKey = "user_id"
	emailKey  ctxKey = "email"
	roleKey   ctxKey = "role"
)

// Authenticate validates a token string and returns a context populated with user
// identity. Returns (ctx, false) when validation fails.
func Authenticate(ctx context.Context, ts services.TokenService, token string) (context.Context, bool) {
	claims, err := ts.Validate(token)
	if err != nil || claims == nil {
		slog.Debug("auth: token validation failed", "error", err)
		return ctx, false
	}
	ctx = context.WithValue(ctx, userIDKey, claims.UserID)
	ctx = context.WithValue(ctx, emailKey, claims.Email)
	ctx = context.WithValue(ctx, roleKey, string(claims.Role))
	return ctx, true
}

// ExtractToken returns the bearer/cookie token from the request. Cookie takes
// priority over the Authorization header to match prior behaviour.
func ExtractToken(r *http.Request) string {
	if c, err := r.Cookie("session"); err == nil && c.Value != "" {
		return c.Value
	}
	if bearer := r.Header.Get("Authorization"); strings.HasPrefix(bearer, "Bearer ") {
		return strings.TrimPrefix(bearer, "Bearer ")
	}
	return ""
}

// RequireAuth wraps a handler with token validation. Used for endpoints outside
// the ogen-generated server (SSE).
func RequireAuth(ts services.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ExtractToken(r)
			if token == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			ctx, ok := Authenticate(r.Context(), ts, token)
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user ID set by Authenticate.
func UserIDFromContext(ctx context.Context) (uint, bool) {
	v, ok := ctx.Value(userIDKey).(uint)
	return v, ok
}

// RoleFromContext returns the authenticated role set by Authenticate.
func RoleFromContext(ctx context.Context) (models.Role, bool) {
	s, ok := ctx.Value(roleKey).(string)
	if !ok {
		return "", false
	}
	return models.Role(s), true
}

// EmailFromContext returns the authenticated email set by Authenticate.
func EmailFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(emailKey).(string)
	return v, ok
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
