package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEHub_SubscribeAndBroadcast(t *testing.T) {
	hub := NewSSEHub()
	ch := hub.Subscribe(42)
	defer hub.Unsubscribe(ch)

	hub.Broadcast(SSEEvent{Type: "test.event", DocumentID: 42, Payload: map[string]any{"key": "val"}})

	select {
	case msg := <-ch:
		assert.Contains(t, msg, "test.event")
		assert.Contains(t, msg, "data: ")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("did not receive broadcast")
	}
}

func TestSSEHub_FilterByDocumentID(t *testing.T) {
	hub := NewSSEHub()
	ch1 := hub.Subscribe(1)
	ch2 := hub.Subscribe(2)
	defer hub.Unsubscribe(ch1)
	defer hub.Unsubscribe(ch2)

	hub.Broadcast(SSEEvent{Type: "doc.updated", DocumentID: 1})

	select {
	case <-ch1:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch1 should receive event for doc 1")
	}

	select {
	case <-ch2:
		t.Fatal("ch2 should NOT receive event for doc 1")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestSSEHub_WildcardSubscriber(t *testing.T) {
	hub := NewSSEHub()
	// documentID=0 means subscribe to all events
	ch := hub.Subscribe(0)
	defer hub.Unsubscribe(ch)

	hub.Broadcast(SSEEvent{Type: "any.event", DocumentID: 99})

	select {
	case msg := <-ch:
		assert.Contains(t, msg, "any.event")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("wildcard subscriber should receive all events")
	}
}

func TestSSEHub_Unsubscribe(t *testing.T) {
	hub := NewSSEHub()
	ch := hub.Subscribe(1)
	hub.Unsubscribe(ch)

	hub.Broadcast(SSEEvent{Type: "after.unsub", DocumentID: 1})

	select {
	case <-ch:
		t.Fatal("unsubscribed client should not receive events")
	case <-time.After(50 * time.Millisecond):
		// expected
	}
}

func TestSSEHub_BroadcastFormat(t *testing.T) {
	hub := NewSSEHub()
	ch := hub.Subscribe(5)
	defer hub.Unsubscribe(ch)

	hub.Broadcast(SSEEvent{Type: "document.approved", DocumentID: 5, Payload: map[string]any{"status": "approved"}})

	msg := <-ch
	// Should be SSE format: "data: {json}\n\n"
	assert.True(t, len(msg) > 6)
	assert.Equal(t, "data: ", msg[:6])
	assert.Equal(t, "\n\n", msg[len(msg)-2:])

	// Parse the JSON payload
	jsonStr := msg[6 : len(msg)-2]
	var event SSEEvent
	err := json.Unmarshal([]byte(jsonStr), &event)
	require.NoError(t, err)
	assert.Equal(t, "document.approved", event.Type)
	assert.Equal(t, uint(5), event.DocumentID)
}

func TestSSEHub_SlowClientDrop(t *testing.T) {
	hub := NewSSEHub()
	ch := hub.Subscribe(1)
	defer hub.Unsubscribe(ch)

	// Fill the buffer (size 16) then send one more
	for i := range 20 {
		hub.Broadcast(SSEEvent{Type: "flood", DocumentID: 1, Payload: i})
	}

	// Should have exactly 16 messages (buffer size)
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	assert.Equal(t, 16, count, "slow client should receive at most buffer size messages")
}

func TestSSEHub_MultipleClients(t *testing.T) {
	hub := NewSSEHub()
	ch1 := hub.Subscribe(1)
	ch2 := hub.Subscribe(1)
	defer hub.Unsubscribe(ch1)
	defer hub.Unsubscribe(ch2)

	hub.Broadcast(SSEEvent{Type: "multi", DocumentID: 1})

	select {
	case <-ch1:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch1 should receive")
	}
	select {
	case <-ch2:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch2 should receive")
	}
}
