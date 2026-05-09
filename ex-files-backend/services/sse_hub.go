package services

import (
	"encoding/json"
	"sync"
)

type SSEEvent struct {
	Type       string `json:"type"` // e.g. "document.approved", "comment.added"
	DocumentID uint   `json:"documentId"`
	// UserID, when non-zero, restricts delivery to subscribers whose userID
	// matches. Used by review-feedback / deadline-reminder notifications so
	// only the affected user gets the toast. Not serialised - the client has
	// no business seeing who else the event was intended for.
	UserID  uint `json:"-"`
	Payload any  `json:"payload,omitempty"`
}

type sseClient struct {
	ch         chan string
	documentID uint // 0 means subscribe to events for any document
	userID     uint // 0 means caller wants only untargeted broadcasts
}

type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan string]sseClient
}

func NewSSEHub() *SSEHub {
	return &SSEHub{clients: make(map[chan string]sseClient)}
}

// Subscribe registers a client. documentID=0 receives events for any document;
// userID is the authenticated user - events tagged with a different UserID are
// not delivered to this client. Pass userID=0 only for service-internal
// subscribers; never trust a client-supplied userID over an authenticated one.
func (h *SSEHub) Subscribe(documentID, userID uint) chan string {
	ch := make(chan string, 16)
	h.mu.Lock()
	h.clients[ch] = sseClient{ch: ch, documentID: documentID, userID: userID}
	h.mu.Unlock()
	return ch
}

func (h *SSEHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// Broadcast delivers an event to every subscriber it concerns:
//   - documentID match: client.documentID == 0 || client.documentID == event.DocumentID
//   - user match:       event.UserID == 0  || client.userID == event.UserID
//
// A targeted event (event.UserID != 0) reaches only matching authenticated
// clients - clients subscribed with userID=0 don't snoop on other users'
// targeted events.
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
		if event.UserID != 0 && client.userID != event.UserID {
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
