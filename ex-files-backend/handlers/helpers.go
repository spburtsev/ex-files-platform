package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"github.com/spburtsev/ex-files-backend/models"
)

// mustGetUserID extracts the authenticated user's ID from the Gin context.
// Returns false and aborts the request if the value is missing or has the wrong type.
func mustGetUserID(c *gin.Context) (uint, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return 0, false
	}
	uid, ok := v.(uint)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return 0, false
	}
	return uid, true
}

// mustGetRole extracts the authenticated user's role from the Gin context.
// Returns false and aborts the request if the value is missing or has the wrong type.
func mustGetRole(c *gin.Context) (models.Role, bool) {
	v, exists := c.Get("role")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing role context"})
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid role context"})
		return "", false
	}
	return models.Role(s), true
}

const defaultPageSize = 20

func parsePagination(c *gin.Context) (page, perPage int) {
	page = 1
	perPage = defaultPageSize

	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.Query("per_page")); err == nil && v > 0 && v <= 100 {
		perPage = v
	}
	return
}

func setPaginationHeaders(c *gin.Context, page, perPage int, total int64) {
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	c.Header("X-Total-Count", fmt.Sprintf("%d", total))
	c.Header("X-Page", fmt.Sprintf("%d", page))
	c.Header("X-Per-Page", fmt.Sprintf("%d", perPage))
	c.Header("X-Total-Pages", fmt.Sprintf("%d", totalPages))
}

func parseTime(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format: %s", s)
}

func protobufResponse(c *gin.Context, status int, msg proto.Message) {
	b, err := proto.Marshal(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "serialization failed"})
		return
	}
	c.Data(status, "application/x-protobuf", b)
}
