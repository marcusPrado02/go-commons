package outbox_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/app/outbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inMemoryStore is a test double for OutboxStore.
type inMemoryStore struct {
	mu        sync.Mutex
	messages  []outbox.OutboxMessage
	processed map[string]bool
}

func newInMemoryStore() *inMemoryStore {
	return &inMemoryStore{processed: make(map[string]bool)}
}

func (s *inMemoryStore) Save(_ context.Context, msgs []outbox.OutboxMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msgs...)
	return nil
}

func (s *inMemoryStore) FetchPending(_ context.Context, limit int) ([]outbox.OutboxMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pending []outbox.OutboxMessage
	for _, m := range s.messages {
		if !s.processed[m.ID] {
			pending = append(pending, m)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (s *inMemoryStore) MarkProcessed(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processed[id] = true
	return nil
}

// inMemoryPublisher records published messages.
type inMemoryPublisher struct {
	mu        sync.Mutex
	published []outbox.OutboxMessage
}

func (p *inMemoryPublisher) Publish(_ context.Context, msg outbox.OutboxMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.published = append(p.published, msg)
	return nil
}

func TestOutboxMessage_IDIsIdempotentKey(t *testing.T) {
	msg := outbox.OutboxMessage{
		ID:          "msg-1",
		AggregateID: "order-42",
		EventType:   "OrderPlaced",
		Payload:     []byte(`{"orderId":"42"}`),
		CreatedAt:   time.Now(),
	}
	assert.Equal(t, "msg-1", msg.ID)
}

func TestOutboxPublisher_ProcessesPendingMessages(t *testing.T) {
	store := newInMemoryStore()
	pub := &inMemoryPublisher{}

	_ = store.Save(context.Background(), []outbox.OutboxMessage{
		{ID: "1", EventType: "OrderPlaced", Payload: []byte(`{}`), CreatedAt: time.Now()},
		{ID: "2", EventType: "OrderShipped", Payload: []byte(`{}`), CreatedAt: time.Now()},
	})

	publisher := outbox.NewPublisher(store, pub.Publish,
		outbox.WithBatchSize(10),
		outbox.WithPollingInterval(10*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := publisher.Start(ctx)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	_ = publisher.Stop(context.Background())

	pub.mu.Lock()
	defer pub.mu.Unlock()
	assert.Len(t, pub.published, 2)
}

func TestOutboxPublisher_MarkProcessed_IsIdempotent(t *testing.T) {
	store := newInMemoryStore()
	_ = store.Save(context.Background(), []outbox.OutboxMessage{
		{ID: "1", EventType: "E", Payload: []byte(`{}`), CreatedAt: time.Now()},
	})

	// MarkProcessed twice should not error
	err := store.MarkProcessed(context.Background(), "1")
	require.NoError(t, err)
	err = store.MarkProcessed(context.Background(), "1")
	require.NoError(t, err)

	pending, _ := store.FetchPending(context.Background(), 10)
	assert.Empty(t, pending)
}

func TestOutboxPublisher_Start_IsNonBlocking(t *testing.T) {
	store := newInMemoryStore()
	pub := &inMemoryPublisher{}
	publisher := outbox.NewPublisher(store, pub.Publish)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = publisher.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Start returned quickly — non-blocking confirmed
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Start blocked for too long — should be non-blocking")
	}
}
