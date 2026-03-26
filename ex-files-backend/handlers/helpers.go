package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

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

func protobufResponse(c *gin.Context, status int, msg proto.Message) {
	b, err := proto.Marshal(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "serialization failed"})
		return
	}
	c.Data(status, "application/x-protobuf", b)
}
