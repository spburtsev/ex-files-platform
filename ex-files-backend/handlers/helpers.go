package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

func protobufResponse(c *gin.Context, status int, msg proto.Message) {
	b, err := proto.Marshal(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "serialization failed"})
		return
	}
	c.Data(status, "application/x-protobuf", b)
}
