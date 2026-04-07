package queue_test

import (
	"context"
	"testing"

	"github.com/marcusPrado02/go-commons/ports/queue"
)

// Compile-time check that QueuePort can be implemented.
var _ queue.QueuePort = (*nilQueue)(nil)

type nilQueue struct{}

func (n *nilQueue) Publish(_ context.Context, _ string, _ queue.Message) error { return nil }
func (n *nilQueue) Subscribe(_ context.Context, _ string, _ queue.Handler) (func(), error) {
	return func() {}, nil
}
func (n *nilQueue) Ping(_ context.Context) error { return nil }

func TestQueueMessage_Fields(t *testing.T) {
	msg := queue.Message{
		ID:         "msg-1",
		Topic:      "orders",
		Payload:    []byte(`{"id":1}`),
		Attributes: map[string]string{"source": "api"},
	}
	if msg.ID != "msg-1" {
		t.Errorf("expected ID %q, got %q", "msg-1", msg.ID)
	}
	if msg.Topic != "orders" {
		t.Errorf("expected Topic %q, got %q", "orders", msg.Topic)
	}
	if string(msg.Payload) != `{"id":1}` {
		t.Errorf("unexpected Payload: %q", msg.Payload)
	}
	if msg.Attributes["source"] != "api" {
		t.Errorf("expected Attributes[source]=%q, got %q", "api", msg.Attributes["source"])
	}
}

func TestQueueMessage_ZeroValue(t *testing.T) {
	var msg queue.Message
	if msg.ID != "" || msg.Topic != "" || msg.Payload != nil || msg.Attributes != nil {
		t.Fatal("expected all-zero Message fields")
	}
}

func TestHandler_IsFunc(t *testing.T) {
	// Handler must be a func type; verify it can be assigned from a closure.
	var h queue.Handler = func(_ context.Context, _ queue.Message) error { return nil }
	if h == nil {
		t.Fatal("expected non-nil Handler")
	}
}
