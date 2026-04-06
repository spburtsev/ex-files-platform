package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/spburtsev/ex-files-backend/services"
)

func AuthMiddleware(ts services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		var source string

		if cookie, err := c.Cookie("session"); err == nil {
			tokenStr = cookie
			source = "cookie"
		} else if bearer := c.GetHeader("Authorization"); strings.HasPrefix(bearer, "Bearer ") {
			tokenStr = strings.TrimPrefix(bearer, "Bearer ")
			source = "bearer"
		}

		if tokenStr == "" {
			slog.Debug("auth: no token found", "path", c.Request.URL.Path, "method", c.Request.Method)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := ts.Validate(tokenStr)
		if err != nil || claims == nil {
			slog.Debug("auth: token validation failed", "source", source, "error", err, "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		slog.Debug("auth: authenticated", "user_id", claims.UserID, "email", claims.Email, "role", claims.Role, "source", source)
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", string(claims.Role))
		c.Next()
	}
}
