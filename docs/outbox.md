# Transactional Outbox Pattern

The Transactional Outbox solves the **dual-write problem**: how to atomically update your database *and* publish a message to a broker without distributed transactions.

## The Problem

```
// UNSAFE — two separate writes, one can fail:
repo.Save(ctx, order)          // 1. write to DB
broker.Publish(ctx, "orders", msg) // 2. write to broker
```

If step 2 fails, the order is saved but the event is never delivered. If step 1 fails after step 2 succeeds, you have a phantom event.

## The Solution

Persist the message **inside the same database transaction** as the aggregate. A background publisher polls the outbox table and delivers pending messages.

```
Transaction {
    repo.Save(ctx, order)          // write aggregate
    outboxStore.Save(ctx, msgs)    // write message — same TX
}
// Background:
publisher.Start(ctx)               // polls and delivers
```

This gives you **at-least-once delivery** with idempotency via message ID.

---

## Usage

### 1. Implement OutboxStore

`OutboxStore` is a port — you implement it backed by your application database.

```go
type pgOutboxStore struct{ db *sql.DB }

func (s *pgOutboxStore) Save(ctx context.Context, msgs []outbox.OutboxMessage) error {
    // INSERT into outbox_messages within the caller's transaction
}

func (s *pgOutboxStore) FetchPending(ctx context.Context, limit int) ([]outbox.OutboxMessage, error) {
    // SELECT ... WHERE processed_at IS NULL ORDER BY created_at LIMIT $1
}

func (s *pgOutboxStore) MarkProcessed(ctx context.Context, id string) error {
    // UPDATE outbox_messages SET processed_at = NOW() WHERE id = $1
}
```

### 2. Create and Start the Publisher

```go
import "github.com/marcusPrado02/go-commons/app/outbox"

publisher := outbox.NewPublisher(
    store,
    func(ctx context.Context, msg outbox.OutboxMessage) error {
        return broker.Publish(ctx, msg.EventType, queue.Message{
            ID:      msg.ID,
            Payload: msg.Payload,
        })
    },
    outbox.WithPollingInterval(5*time.Second),
    outbox.WithBatchSize(50),
    outbox.WithConcurrency(2),
    outbox.WithLogger(logger),
)

if err := publisher.Start(ctx); err != nil {
    log.Fatal(err)
}
defer publisher.Stop(shutdownCtx)
```

### 3. Save Messages in the Same Transaction

```go
func (s *OrderService) PlaceOrder(ctx context.Context, order Order) error {
    msgs := []outbox.OutboxMessage{{
        ID:          uuid.NewString(),
        AggregateID: order.ID,
        EventType:   "order.placed",
        Payload:     mustMarshal(OrderPlacedEvent{OrderID: order.ID}),
        CreatedAt:   time.Now(),
    }}

    return s.db.WithTx(ctx, func(ctx context.Context) error {
        if _, err := s.repo.Save(ctx, order); err != nil {
            return err
        }
        return s.outboxStore.Save(ctx, msgs)
    })
}
```

---

## Configuration Reference

| Option | Default | Description |
|---|---|---|
| `WithPollingInterval(d)` | 5s | How often to poll for pending messages |
| `WithBatchSize(n)` | 100 | Max messages per polling cycle |
| `WithConcurrency(n)` | 1 | Concurrent delivery goroutines (1 = ordered) |
| `WithLogger(l)` | nil | Logger for delivery errors and lifecycle events |

---

## Delivery Guarantees

- **At-least-once**: A message may be delivered more than once if the publisher restarts between delivery and `MarkProcessed`. Make your consumers idempotent using the message `ID` as a deduplication key.
- **No ordering guarantee across batches**: Within a single batch, messages are processed in `FetchPending` order. Across batches, ordering is not guaranteed if `WithConcurrency > 1`.
- **No dead-letter queue**: Messages that continuously fail to deliver are retried indefinitely. Add a max-attempts check in your `FetchPending` implementation if you need a DLQ.

---

## Idempotency Key

`OutboxMessage.ID` is the idempotency key. Always use a UUID generated at the time of the business event — never use database sequence IDs as they are not globally unique across services.

```go
msgs := []outbox.OutboxMessage{{
    ID:        uuid.NewString(), // stable across retries
    EventType: "order.placed",
    Payload:   payload,
}}
```

---

## Observability

With `WithLogger` configured, the publisher logs:

- `outbox: fetch pending failed` — store read error
- `outbox: publish failed` — broker delivery error, message left for retry
- `outbox: mark processed failed` — store write error after successful delivery (message will be redelivered)

Instrument your `OutboxStore.FetchPending` with a metric counting `attempts` per message to detect stuck messages.
