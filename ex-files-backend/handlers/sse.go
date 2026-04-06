package handlers

import (
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spburtsev/ex-files-backend/services"
)

type SSEHandler struct {
	Hub *services.SSEHub
}

func (h *SSEHandler) Stream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var documentID uint
	if v, err := strconv.ParseUint(c.Query("documentId"), 10, 64); err == nil {
		documentID = uint(v)
	}

	ch := h.Hub.Subscribe(documentID)
	defer h.Hub.Unsubscribe(ch)

	c.Stream(func(w io.Writer) bool {
		select {
		case msg := <-ch:
			fmt.Fprint(w, msg)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
