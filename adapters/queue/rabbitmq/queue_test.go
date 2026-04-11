package rabbitmq_test

import (
	"context"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/marcusPrado02/go-commons/adapters/queue/rabbitmq"
	"github.com/marcusPrado02/go-commons/ports/queue"
)

// amqpURL returns the broker URL from the environment, defaulting to localhost.
func amqpURL() string {
	if u := os.Getenv("AMQP_URL"); u != "" {
		return u
	}
	return "amqp://guest:guest@localhost:5672/"
}

// skipIfNoBroker dials the broker and skips the test if it is unreachable.
func skipIfNoBroker(t *testing.T) *amqp.Connection {
	t.Helper()
	conn, err := amqp.Dial(amqpURL())
	if err != nil {
		t.Skipf("RabbitMQ not available (%v) — skipping integration test", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// uniqueQueue returns a queue name unique per test to avoid cross-test pollution.
func uniqueQueue(t *testing.T) string {
	return "test-" + t.Name()
}

// --- compile-time interface check -----------------------------------------

var _ queue.Port = (*rabbitmq.Client)(nil)

// --- unit tests -----------------------------------------------------------

func TestToQueueMessage_Fields(t *testing.T) {
	// toQueueMessage is unexported, so we exercise it end-to-end via Subscribe.
	// This test verifies that Payload and Attributes round-trip correctly.
	conn := skipIfNoBroker(t)
	client := rabbitmq.New(conn)
	topic := uniqueQueue(t)

	want := queue.Message{
		Payload:    []byte(`{"hello":"world"}`),
		Attributes: map[string]string{"content-type": "application/json"},
	}
	if err := client.Publish(context.Background(), topic, want); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	received := make(chan queue.Message, 1)
	cancel, err := client.Subscribe(context.Background(), topic, func(_ context.Context, msg queue.Message) error {
		received <- msg
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer cancel()

	select {
	case got := <-received:
		if string(got.Payload) != string(want.Payload) {
			t.Errorf("Payload: got %q, want %q", got.Payload, want.Payload)
		}
		if got.Attributes["content-type"] != want.Attributes["content-type"] {
			t.Errorf("Attribute content-type: got %q, want %q",
				got.Attributes["content-type"], want.Attributes["content-type"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

// --- integration tests ----------------------------------------------------

func TestRabbitMQ_Ping(t *testing.T) {
	conn := skipIfNoBroker(t)
	client := rabbitmq.New(conn)

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestRabbitMQ_Publish(t *testing.T) {
	conn := skipIfNoBroker(t)
	client := rabbitmq.New(conn)
	topic := uniqueQueue(t)

	msg := queue.Message{
		ID:      "msg-123",
		Payload: []byte(`hello rabbitmq`),
	}
	if err := client.Publish(context.Background(), topic, msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func TestRabbitMQ_Subscribe_ReceivesAndAcks(t *testing.T) {
	conn := skipIfNoBroker(t)
	client := rabbitmq.New(conn)
	topic := uniqueQueue(t)

	payload := []byte("integration payload")
	if err := client.Publish(context.Background(), topic, queue.Message{Payload: payload}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	received := make(chan []byte, 1)
	cancel, err := client.Subscribe(context.Background(), topic, func(_ context.Context, msg queue.Message) error {
		received <- msg.Payload
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer cancel()

	select {
	case got := <-received:
		if string(got) != string(payload) {
			t.Errorf("got %q, want %q", got, payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestRabbitMQ_Subscribe_NacksOnHandlerError(t *testing.T) {
	conn := skipIfNoBroker(t)
	client := rabbitmq.New(conn)
	topic := uniqueQueue(t)

	if err := client.Publish(context.Background(), topic, queue.Message{Payload: []byte("nack-me")}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// First handler fails — message should be requeued.
	attempts := make(chan struct{}, 10)
	handlerCalls := 0
	cancel, err := client.Subscribe(context.Background(), topic, func(_ context.Context, _ queue.Message) error {
		handlerCalls++
		attempts <- struct{}{}
		if handlerCalls == 1 {
			return context.DeadlineExceeded // simulate failure
		}
		return nil // second attempt succeeds
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer cancel()

	// Wait for at least two deliveries (nack + requeue + ack).
	deadline := time.After(10 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-attempts:
		case <-deadline:
			t.Fatalf("timeout: only %d handler call(s) observed, expected 2", handlerCalls)
		}
	}
}

func TestRabbitMQ_Ping_ClosedConnection(t *testing.T) {
	conn := skipIfNoBroker(t)
	conn.Close() // close before ping
	client := rabbitmq.New(conn)

	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("expected error from Ping on closed connection, got nil")
	}
}

func TestRabbitMQ_Publish_WithAttributes(t *testing.T) {
	conn := skipIfNoBroker(t)
	client := rabbitmq.New(conn)
	topic := uniqueQueue(t)

	msg := queue.Message{
		Payload: []byte("attrs test"),
		Attributes: map[string]string{
			"trace-id":  "abc-123",
			"source-svc": "auth",
		},
	}
	if err := client.Publish(context.Background(), topic, msg); err != nil {
		t.Fatalf("Publish with attributes: %v", err)
	}

	received := make(chan queue.Message, 1)
	cancel, err := client.Subscribe(context.Background(), topic, func(_ context.Context, m queue.Message) error {
		received <- m
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer cancel()

	select {
	case got := <-received:
		if got.Attributes["trace-id"] != "abc-123" {
			t.Errorf("trace-id: got %q, want %q", got.Attributes["trace-id"], "abc-123")
		}
		if got.Attributes["source-svc"] != "auth" {
			t.Errorf("source-svc: got %q, want %q", got.Attributes["source-svc"], "auth")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message with attributes")
	}
}
