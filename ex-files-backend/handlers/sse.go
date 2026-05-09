package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/spburtsev/ex-files-backend/middleware"
	"github.com/spburtsev/ex-files-backend/services"
)

// SSEHandler streams Server-Sent Events to authenticated clients. It sits
// outside the ogen-generated handler because OpenAPI does not model
// text/event-stream cleanly.
type SSEHandler struct {
	Hub *services.SSEHub
}

func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// /events is mounted behind RequireAuth, so the user ID is in context.
	// We deliberately ignore any client-supplied userId query param, that's
	// how we keep one user from snooping on another's targeted events.
	uid, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var documentID uint
	if v, err := strconv.ParseUint(r.URL.Query().Get("documentId"), 10, 64); err == nil {
		documentID = uint(v)
	}

	ch := h.Hub.Subscribe(documentID, uid)
	defer h.Hub.Unsubscribe(ch)

	for {
		select {
		case msg := <-ch:
			if _, err := fmt.Fprint(w, msg); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
