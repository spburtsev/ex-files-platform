package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger returns Gin middleware that logs every request as structured JSON.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", float64(time.Since(start).Microseconds()) / 1000.0,
			"client_ip", c.ClientIP(),
		}

		if uid, ok := c.Get("userID"); ok {
			attrs = append(attrs, "user_id", uid)
		}

		if origin := c.GetHeader("Origin"); origin != "" {
			attrs = append(attrs, "origin", origin)
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.String())
		}

		switch {
		case status >= 500:
			slog.Error("request", attrs...)
		case status >= 400:
			slog.Warn("request", attrs...)
		default:
			slog.Info("request", attrs...)
		}
	}
}
