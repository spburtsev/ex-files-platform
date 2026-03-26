package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

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
