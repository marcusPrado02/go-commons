# go-commons

A shared Go library implementing **Hexagonal Architecture** building blocks: ports (interfaces), adapters (implementations), and application-layer services. Designed to be composed — use only what you need.

---

## Feature Matrix

### Kernel (zero external dependencies)

| Package | Description |
|---|---|
| `kernel/errors` | `Problem`, `ErrorCode`, `DomainError` — rich, immutable domain errors |
| `kernel/result` | `Result[T]` — functional pipelines with `Map`, `FlatMap`, `Or`, `OrElse` |
| `kernel/ddd` | `AggregateRoot`, `DomainEvent` — DDD base types |

### Ports (interface contracts)

| Package | Interface | Key Methods |
|---|---|---|
| `ports/cache` | `CachePort` | `Get`, `Set`, `Delete`, `Exists` |
| `ports/email` | `EmailPort` | `Send`, `SendWithTemplate`, `Ping` |
| `ports/files` | `FileStorePort` | `Upload`, `Download`, `Delete`, `List`, `GeneratePresignedURL` |
| `ports/queue` | `QueuePort` | `Publish`, `Subscribe`, `Ping` |
| `ports/push` | `PushPort` | `Send`, `Ping` |
| `ports/sms` | `SMSPort` | `Send`, `Ping` |
| `ports/secrets` | `SecretsPort` | `Get`, `GetJSON` |
| `ports/compression` | `CompressionPort` | `Compress`, `Decompress` |
| `ports/excel` | `ExcelPort` | `Generate` |
| `ports/persistence` | `Repository[E,ID]`, `PageableRepository[E,ID]` | CRUD + paginated `FindAll`/`Search` |
| `ports/transaction` | `TransactionManager` | `Begin`, `Commit`, `Rollback`, `WithTx` |
| `ports/ratelimit` | `RateLimiter` | `Allow`, `Wait` |
| `ports/featureflag` | `FeatureFlagPort` | `IsEnabled` |
| `ports/audit` | `AuditLog` | `Record` |
| `ports/eventbus` | `EventBus` | `Publish`, `Subscribe` |
| `ports/observability` | `Logger`, `Tracer` | Structured logging and tracing |

### Application Layer

| Package | Description |
|---|---|
| `app/outbox` | Transactional Outbox pattern — at-least-once async delivery |
| `app/resilience` | Retry with jitter backoff + circuit breaker (`gobreaker`) |
| `app/scheduler` | Cron-based job scheduler with panic recovery |
| `app/observability` | Health checks (liveness/readiness) + log sanitizer |

### Adapters (each is an independent Go module)

| Adapter | Port | Notes |
|---|---|---|
| `adapters/cache/redis` | `CachePort` | `go-redis/v9`; test with `miniredis` |
| `adapters/email/smtp` | `EmailPort` | stdlib `net/smtp`, TLS |
| `adapters/email/ses` | `EmailPort` | AWS SES v1 (no template support) |
| `adapters/email/sesv2` | `EmailPort` | AWS SES v2, full template support |
| `adapters/email/sendgrid` | `EmailPort` | SendGrid v3, built-in retry on 429/5xx |
| `adapters/files/s3` | `FileStorePort` | AWS S3, presign, SSE |
| `adapters/files/gcs` | `FileStorePort` | Google Cloud Storage |
| `adapters/queue/sqs` | `QueuePort` | AWS SQS, long-polling |
| `adapters/queue/rabbitmq` | `QueuePort` | AMQP 0-9-1, durable queues |
| `adapters/push/fcm` | `PushPort` | Firebase Cloud Messaging |
| `adapters/sms/twilio` | `SMSPort` | Twilio REST API |
| `adapters/payment/stripe` | — | PaymentIntent, Refund, Webhook verification |
| `adapters/secrets/awsssm` | `SecretsPort` | AWS SSM Parameter Store, SecureString |
| `adapters/secrets/vault` | `SecretsPort` | HashiCorp Vault KV v2, stdlib only |
| `adapters/compression/stdlib` | `CompressionPort` | gzip + flate, no external deps |
| `adapters/search/elasticsearch` | — | Elasticsearch 8.x |
| `adapters/search/opensearch` | — | OpenSearch |
| `adapters/tracing/otel` | `Tracer` | OpenTelemetry |
| `adapters/persistence/inmemory` | `PageableRepository` | Thread-safe, TTL, test-friendly |

### Testkit

| Package | Description |
|---|---|
| `testkit/assert` | `AggregateAssert` — DDD event assertion helpers |
| `testkit/contracts` | Reusable contract test suites for all major ports |

---

## Quick Start

```bash
# Root module (kernel + ports + app layer)
go get github.com/marcusPrado02/go-commons

# Individual adapters (each is a separate module)
go get github.com/marcusPrado02/go-commons/adapters/cache/redis
go get github.com/marcusPrado02/go-commons/adapters/queue/sqs
```

### Minimal example — send an email

```go
import (
    emailport "github.com/marcusPrado02/go-commons/ports/email"
    "github.com/marcusPrado02/go-commons/adapters/email/smtp"
)

from, _ := emailport.NewEmailAddress("app@example.com")
client := smtp.New("smtp.example.com", 465, "user", "pass", from)

receipt, err := client.Send(ctx, emailport.Email{
    From:    from,
    To:      []emailport.EmailAddress{to},
    Subject: "Hello",
    HTML:    "<p>World</p>",
})
```

### Minimal example — publish to a queue

```go
import (
    "github.com/marcusPrado02/go-commons/adapters/queue/rabbitmq"
    "github.com/marcusPrado02/go-commons/ports/queue"
    amqp "github.com/rabbitmq/amqp091-go"
)

conn, _ := amqp.Dial("amqp://guest:guest@localhost:5672/")
client := rabbitmq.New(conn)
client.Publish(ctx, "orders", queue.Message{Payload: []byte(`{"id":1}`)})
```

See [`examples/`](examples/) for complete working applications.

---

## Running Locally

```bash
# Start all infrastructure dependencies
docker compose up -d

# Run all tests (integration tests auto-skip if infra is unavailable)
make test

# Run linter
make lint

# Check for vulnerabilities
make vulncheck
```

---

## Package Links

- [Architecture](docs/architecture.md)
- [Error Handling](docs/error-handling.md)
- [Outbox Pattern](docs/outbox.md)
- [Resilience](docs/resilience.md)
- [Adapter Selection](docs/adapter-selection.md)
- [Contributing](docs/contributing.md)
