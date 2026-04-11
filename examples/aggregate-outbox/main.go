// Example: Aggregate + Result + Outbox
//
// This example demonstrates:
//   - Defining a DDD aggregate with domain events
//   - Using Result[T] to chain fallible operations
//   - Persisting messages via the Transactional Outbox pattern
//
// Run: go run ./examples/aggregate-outbox/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/marcusPrado02/go-commons/app/outbox"
	"github.com/marcusPrado02/go-commons/kernel/ddd"
	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
	"github.com/marcusPrado02/go-commons/kernel/result"
	"github.com/marcusPrado02/go-commons/ports/queue"
)

// --- Domain Events ----------------------------------------------------------

// OrderPlacedEvent is emitted when an order is successfully placed.
type OrderPlacedEvent struct {
	OrderID  string    `json:"order_id"`
	Amount   int       `json:"amount_cents"`
	PlacedAt time.Time `json:"placed_at"`

	occurredAt time.Time
}

func (e OrderPlacedEvent) OccurredAt() time.Time { return e.occurredAt }
func (e OrderPlacedEvent) EventType() string      { return "order.placed" }

// --- Domain Model -----------------------------------------------------------

// Order is a DDD aggregate with a string ID.
type Order struct {
	ddd.AggregateRoot[string]
	Amount int
}

// Place creates a new Order and records an OrderPlaced domain event.
func Place(id string, amount int) result.Result[*Order] {
	if amount <= 0 {
		return result.Fail[*Order](kerrors.ErrValidation.
			WithDetail("field", "amount").
			WithDetail("reason", "must be positive"))
	}
	o := &Order{
		AggregateRoot: ddd.NewAggregateRoot(id),
		Amount:        amount,
	}
	o.RegisterEvent(OrderPlacedEvent{
		OrderID:    id,
		Amount:     amount,
		PlacedAt:   time.Now(),
		occurredAt: time.Now(),
	})
	return result.Ok(o)
}

// --- In-Memory Outbox Store -------------------------------------------------

type memOutboxStore struct {
	pending   []outbox.Message
	processed map[string]bool
}

func newMemStore() *memOutboxStore {
	return &memOutboxStore{processed: make(map[string]bool)}
}

func (s *memOutboxStore) Save(_ context.Context, msgs []outbox.Message) error {
	s.pending = append(s.pending, msgs...)
	return nil
}

func (s *memOutboxStore) FetchPending(_ context.Context, limit int) ([]outbox.Message, error) {
	var out []outbox.Message
	for _, m := range s.pending {
		if !s.processed[m.ID] {
			out = append(out, m)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *memOutboxStore) MarkProcessed(_ context.Context, id string) error {
	s.processed[id] = true
	return nil
}

// --- Main -------------------------------------------------------------------

func main() {
	ctx := context.Background()
	store := newMemStore()

	// 1. Place an order using Result[T].
	orderResult := Place("ord-001", 4999)
	order, err := orderResult.Unwrap()
	if err != nil {
		log.Fatal("place order:", err)
	}
	fmt.Printf("Order placed: %s, amount: %d cents\n", order.ID(), order.Amount)

	// 2. Pull domain events and convert to outbox messages (simulating a transaction).
	events := order.PullDomainEvents()
	msgs := make([]outbox.Message, 0, len(events))
	for _, evt := range events {
		payload, _ := json.Marshal(evt)
		msgs = append(msgs, outbox.Message{
			ID:          fmt.Sprintf("%s-%s", order.ID(), evt.EventType()),
			AggregateID: order.ID(),
			EventType:   evt.EventType(),
			Payload:     payload,
			CreatedAt:   time.Now(),
		})
	}
	if err := store.Save(ctx, msgs); err != nil {
		log.Fatal("save outbox:", err)
	}
	fmt.Printf("Saved %d outbox message(s)\n", len(msgs))

	// 3. Start the outbox publisher — delivers to an in-memory channel.
	delivered := make(chan queue.Message, 10)
	publisher := outbox.NewPublisher(
		store,
		func(ctx context.Context, msg outbox.Message) error {
			delivered <- queue.Message{ID: msg.ID, Payload: msg.Payload}
			return nil
		},
		outbox.WithPollingInterval(50*time.Millisecond),
	)
	if err := publisher.Start(ctx); err != nil {
		log.Fatal("start publisher:", err)
	}

	// 4. Wait for delivery and print the result.
	select {
	case msg := <-delivered:
		fmt.Printf("Delivered message ID: %s\n", msg.ID)
		fmt.Printf("Payload: %s\n", msg.Payload)
	case <-time.After(2 * time.Second):
		log.Fatal("timeout waiting for delivery")
	}

	// 5. Show Result[T] error path.
	badResult := Place("ord-002", -1)
	if badResult.IsFail() {
		fmt.Printf("\nBad order error: %v\n", badResult.Problem())
	}

	stopCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_ = publisher.Stop(stopCtx)
}
