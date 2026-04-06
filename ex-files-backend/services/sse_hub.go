package services

import (
	"encoding/json"
	"sync"
)

type SSEEvent struct {
	Type       string `json:"type"`       // e.g. "document.approved", "comment.added"
	DocumentID uint   `json:"documentId"`
	Payload    any    `json:"payload,omitempty"`
}

type sseClient struct {
	ch         chan string
	documentID uint // 0 means subscribe to all events
}

type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan string]sseClient
}

func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[chan string]sseClient)}
}

// Subscribe registers a client. If documentID is 0, the client receives all events.
func (h *SSEHub) Subscribe(documentID uint) chan string {
	ch := make(chan string, 16)
	h.mu.Lock()
	h.clients[ch] = sseClient{ch: ch, documentID: documentID}
	h.mu.Unlock()
	return ch
}

func (h *SSEHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// Broadcast sends an event only to clients subscribed to the same documentID
// or to clients subscribed to all events (documentID == 0).
func (h *SSEHub) Broadcast(event SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	msg := "data: " + string(data) + "\n\n"
	h.mu.RLock()
	for _, client := range h.clients {
		if client.documentID != 0 && client.documentID != event.DocumentID {
			continue
		}
		select {
		case client.ch <- msg:
		default:
			// drop if client is slow
		}
	}
	h.mu.RUnlock()
}
