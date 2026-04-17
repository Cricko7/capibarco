package realtime

import (
	"context"
	"sync"

	"github.com/petmatch/chat-service/internal/domain/chat"
)

// Hub fans chat events out to local subscribers.
type Hub struct {
	mu            sync.RWMutex
	subscriptions map[string]map[chan chat.Event]struct{}
}

// NewHub creates an in-memory realtime hub.
func NewHub() *Hub {
	return &Hub{subscriptions: make(map[string]map[chan chat.Event]struct{})}
}

// Publish publishes an event to local conversation subscribers.
func (h *Hub) Publish(ctx context.Context, event chat.Event) error {
	key := event.PartitionKey
	if key == "" {
		key = eventConversationID(event)
	}
	h.mu.RLock()
	subscribers := h.subscriptions[key]
	for ch := range subscribers {
		select {
		case ch <- event:
		case <-ctx.Done():
			h.mu.RUnlock()
			return ctx.Err()
		default:
		}
	}
	h.mu.RUnlock()
	return nil
}

// Subscribe subscribes to conversation events until the returned cancel is called.
func (h *Hub) Subscribe(conversationID string) (<-chan chat.Event, func()) {
	ch := make(chan chat.Event, 32)
	h.mu.Lock()
	if h.subscriptions[conversationID] == nil {
		h.subscriptions[conversationID] = make(map[chan chat.Event]struct{})
	}
	h.subscriptions[conversationID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subscriptions[conversationID], ch)
		if len(h.subscriptions[conversationID]) == 0 {
			delete(h.subscriptions, conversationID)
		}
		close(ch)
	}
	return ch, cancel
}

func eventConversationID(event chat.Event) string {
	switch {
	case event.Conversation != nil:
		return event.Conversation.ID
	case event.Message != nil:
		return event.Message.ConversationID
	case event.ReadReceipt != nil:
		return event.ReadReceipt.ConversationID
	case event.Typing != nil:
		return event.Typing.ConversationID
	default:
		return ""
	}
}
