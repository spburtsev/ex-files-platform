package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/spburtsev/ex-files-backend/services"
)

func AuthMiddleware(ts services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		if cookie, err := c.Cookie("session"); err == nil {
			tokenStr = cookie
		} else if bearer := c.GetHeader("Authorization"); strings.HasPrefix(bearer, "Bearer ") {
			tokenStr = strings.TrimPrefix(bearer, "Bearer ")
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := ts.Validate(tokenStr)
		if err != nil || claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", string(claims.Role))
		c.Next()
	}
}
