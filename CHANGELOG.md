# Changelog

All notable changes to go-commons are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.1.0] - 2026-04-10

### Added

#### Kernel
- `kernel/errors`: `Problem` (immutable value-type domain error with `WithDetail`, `WithDetails`, `WithCause`), `ErrorCode`, `DomainError` interface, sentinel errors (`ErrNotFound`, `ErrValidation`, `ErrTechnical`, `ErrUnauthorized`)
- `kernel/result`: Generic `Result[T]` with `Ok`, `Fail`, `FromError`, `Map`, `FlatMap`, `Or`, `OrElse`, `Unwrap`
- `kernel/ddd`: `AggregateRoot[ID]`, `DomainEvent` interface, `NewAggregateRoot`, `RegisterEvent`, `PullDomainEvents`

#### Ports
- `ports/cache`: `CachePort` — `Get`, `Set`, `Delete`, `Exists`
- `ports/email`: `EmailPort` — `Send`, `SendWithTemplate`, `Ping`; `Email`, `EmailAddress`, `TemplateEmailRequest`, `EmailReceipt`
- `ports/files`: `FileStorePort` — `Upload`, `Download`, `Delete`, `DeleteAll`, `Exists`, `GetMetadata`, `List`, `GeneratePresignedURL`, `Copy`; `StorageClass` constants including `StorageClassIntelligentTiering` and `StorageClassColdline`
- `ports/queue`: `QueuePort` — `Publish`, `Subscribe`, `Ping`; `Message`, `Handler`
- `ports/push`: `PushPort` — `Send`, `Ping`; `PushNotification`, `PushReceipt`
- `ports/sms`: `SMSPort` — `Send`, `Ping`
- `ports/secrets`: `SecretsPort` — `Get`, `GetJSON`; `ParseJSON` helper
- `ports/compression`: `CompressionPort` — `Compress`, `Decompress`; `Format` constants (`gzip`, `zstd`, `snappy`)
- `ports/excel`: `ExcelPort` — `Generate`; `Sheet`, `ExcelRequest`
- `ports/persistence`: `Repository[E,ID]`, `PageableRepository[E,ID]`, `Specification[E]`, `Spec` constructor, `PageRequest` with `Validate()`, `PageResult`, `Sort`
- `ports/transaction`: `TransactionManager` — `Begin`, `Commit`, `Rollback`, `WithTx`
- `ports/ratelimit`: `RateLimiter` — `Allow`, `Wait`
- `ports/featureflag`: `FeatureFlagPort` — `IsEnabled`
- `ports/audit`: `AuditLog` — `Record`; `AuditEvent`
- `ports/eventbus`: `EventBus` — `Publish`, `Subscribe`
- `ports/observability`: `Logger`, `Tracer`, `Metrics`, `Field`, `F`, `Err`, `RequestID`, `UserID`
- `ports/template`: `TemplatePort` — `Render`

#### Application Layer
- `app/outbox`: `OutboxPublisher` — at-least-once async delivery via Transactional Outbox pattern; `OutboxStore` interface; options: `WithPollingInterval`, `WithBatchSize`, `WithConcurrency`, `WithLogger`
- `app/resilience`: `ResilienceExecutor` — retry with full jitter backoff + `gobreaker` circuit breaker; `ResiliencePolicySet`, `CircuitBreakerConfig`, `ValidatePolicies`, generic `Supply[T]`
- `app/scheduler`: `Scheduler` — cron-based job scheduler with panic recovery; options: `WithLogger`, `WithErrorHandler`
- `app/observability`: `HealthChecks` (liveness/readiness), `WithCheckTimeout` per-check timeout, `LogSanitizer` with recursive nested map redaction; default sensitive keys: `password`, `token`, `secret`, `api_key`, `private_key`, `access_token`, `refresh_token`, `ssn`, `cnpj`, `rg`

#### Adapters
- `adapters/cache/redis`: `CachePort` via `go-redis/v9`; `New`, `NewFromClient`; test support via `miniredis`
- `adapters/compression/stdlib`: `CompressionPort` via stdlib `compress/gzip` and `compress/flate`; zero external dependencies
- `adapters/email/smtp`: `EmailPort` via stdlib `net/smtp` with TLS; multipart MIME (HTML + text)
- `adapters/email/ses`: `EmailPort` via AWS SES v1; `SendWithTemplate` returns `ErrTechnical` directing to `sesv2`
- `adapters/email/sesv2`: `EmailPort` via AWS SES v2; full `SendWithTemplate` support with Handlebars variables
- `adapters/email/sendgrid`: `EmailPort` via SendGrid v3; automatic retry on 429/5xx with jitter backoff
- `adapters/files/s3`: `FileStorePort` via AWS S3; SSE, presign (GET/PUT/DELETE), multipart upload via s3manager; `NewWithOptions` for LocalStack
- `adapters/files/gcs`: `FileStorePort` via Google Cloud Storage
- `adapters/payment/stripe`: `CreatePaymentIntent`, `Refund`, `VerifyWebhookSignature`
- `adapters/persistence/inmemory`: Thread-safe generic `InMemoryRepository`; `WithTTL`, `WithSortFunc`, `Clear`
- `adapters/push/fcm`: `PushPort` via Firebase Cloud Messaging; `New` (file), `NewFromCredentialsJSON` (bytes)
- `adapters/queue/sqs`: `QueuePort` via AWS SQS; long-polling consumer
- `adapters/queue/rabbitmq`: `QueuePort` via RabbitMQ (`amqp091-go`); durable queues, ack/nack
- `adapters/search/elasticsearch`: Elasticsearch 8.x client
- `adapters/search/opensearch`: OpenSearch 2.x client
- `adapters/secrets/awsssm`: `SecretsPort` via AWS SSM Parameter Store; `WithDecryption: true`
- `adapters/secrets/vault`: `SecretsPort` via HashiCorp Vault KV v2; stdlib-only, no SDK dependency
- `adapters/sms/twilio`: `SMSPort` via Twilio REST API
- `adapters/tracing/otel`: `Tracer` via OpenTelemetry

#### Testkit
- `testkit/assert`: `AggregateAssert` — domain event assertion helpers
- `testkit/contracts`: Reusable contract suites for `EmailPort`, `FileStorePort`, `CachePort`, `SMSPort`, `QueuePort`, `Repository`

#### Security
- Webhook signature verification for Stripe (`adapters/payment/stripe/webhook.go`)
- Recursive map sanitization in `LogSanitizer`
- Extended default sensitive key list

#### Tooling
- `.github/workflows/ci.yml`: `test` (with `-race`), `lint`, `tidy-check`
- `.golangci.yml`: `errcheck`, `govet`, `staticcheck`, `gosec`, `nilnil`, `gocritic`, `cyclop`
- `Makefile`: `test`, `lint`, `fmt`, `bench`, `vulncheck`, `mock`, `coverage`, `coverage-report`, `tidy`, `tidy-all`
- `docker-compose.yml`: Redis, RabbitMQ, LocalStack, Elasticsearch, OpenSearch

#### Documentation
- `docs/architecture.md`: Hexagonal Architecture overview, layer diagram, dependency rules
- `docs/error-handling.md`: `Problem`, `Result[T]`, error propagation guide
- `docs/outbox.md`: Transactional Outbox pattern, at-least-once guarantees, observability
- `docs/resilience.md`: Retry jitter, circuit breaker states, `ResiliencePolicySet` reference
- `docs/adapter-selection.md`: Comparison table for all adapter families
- `docs/contributing.md`: Commit conventions, adding adapters, test standards
- `examples/aggregate-outbox`: DDD aggregate + Result[T] + Outbox
- `examples/scheduler-resilience`: Scheduler + ResilienceExecutor
- `examples/inmemory-repository`: InMemoryRepository + Specification + pagination

### Fixed
- `app/resilience`: `ValidatePolicies` is now called at the start of `Run()` for fail-fast validation
- `ports/persistence`: `PageRequest.Validate()` returns an error for `Size ≤ 0` or `Page < 0`
- `adapters/persistence/inmemory`: `Search` now calls `req.Validate()` before executing queries

[0.1.0]: https://github.com/marcusPrado02/go-commons/releases/tag/v0.1.0
