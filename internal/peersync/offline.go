package peersync

import (
	"sync"

	"github.com/google/uuid"
	"github.com/marcosfelipeeipper/agentboard/internal/server"
)

// OfflineQueue stores messages that couldn't be sent while disconnected.
// Each message gets an idempotency key to prevent duplicate processing on replay.
type OfflineQueue struct {
	mu       sync.Mutex
	messages []queuedMessage
}

type queuedMessage struct {
	IdempotencyKey string
	Message        server.Message
}

func NewOfflineQueue() *OfflineQueue {
	return &OfflineQueue{}
}

func (q *OfflineQueue) Enqueue(msg server.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = append(q.messages, queuedMessage{
		IdempotencyKey: uuid.New().String(),
		Message:        msg,
	})
}

func (q *OfflineQueue) Drain() []server.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	msgs := make([]server.Message, len(q.messages))
	for i, qm := range q.messages {
		msgs[i] = qm.Message
	}
	q.messages = nil
	return msgs
}

func (q *OfflineQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages)
}
