# Architecture

go-commons is structured around **Hexagonal Architecture** (Ports & Adapters), also known as the Clean Architecture. The goal is to keep business logic independent of infrastructure concerns.

## Layer Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        adapters/                            │
│   (infrastructure: AWS, Redis, RabbitMQ, SMTP, Vault…)      │
│                                                             │
│         implements ▼                    implements ▼        │
├──────────────────────┬──────────────────────────────────────┤
│      app/            │           ports/                     │
│  (application layer) │     (interface contracts)            │
│                      │                                      │
│  outbox              │  cache   email   files   queue       │
│  resilience          │  push    sms     secrets persistence  │
│  scheduler           │  audit   eventbus featureflag …      │
│  observability       │                                      │
├──────────────────────┴──────────────────────────────────────┤
│                        kernel/                              │
│        (domain primitives — zero external deps)             │
│                                                             │
│         errors    result    ddd                             │
└─────────────────────────────────────────────────────────────┘
```

## Layers

### kernel/

The innermost layer. Contains only domain primitives with **zero external dependencies**. Everything else may depend on `kernel`; `kernel` depends on nothing.

- `kernel/errors` — `Problem` (immutable rich error), `ErrorCode`, `DomainError` interface, sentinel errors (`ErrNotFound`, `ErrValidation`, `ErrTechnical`, `ErrUnauthorized`)
- `kernel/result` — `Result[T]` for functional pipelines
- `kernel/ddd` — `AggregateRoot`, `DomainEvent` base types for DDD

### ports/

Interface contracts that define **what** a capability does, without specifying **how**. Your application code depends only on these interfaces — never on a concrete adapter.

Each port is a small, focused Go package with a single interface and its associated value types. Ports have no external dependencies beyond `kernel`.

### app/

Application-layer services that implement cross-cutting concerns. These use ports as dependencies and are infrastructure-agnostic.

- `app/outbox` — Transactional Outbox pattern. Persists messages in the same DB transaction as the aggregate, then delivers asynchronously.
- `app/resilience` — Retry with exponential backoff + full jitter + circuit breaker.
- `app/scheduler` — Cron job scheduler with panic recovery and structured logging.
- `app/observability` — Aggregated health checks (liveness/readiness) and log sanitizer.

### adapters/

Concrete implementations of ports. Each adapter is its own **independent Go module** (has its own `go.mod`) to avoid pulling heavy SDK dependencies into projects that don't need them.

Adapters depend on `ports/` but never on other adapters or `app/`.

## Go Workspace

Because each adapter is an independent module, the repository uses a **Go workspace** (`go.work`) to enable cross-module development without publishing. The workspace file is only for local development — consumers get individual modules via `go get`.

```
go.work
├── .                                  ← root module (kernel + ports + app + testkit)
├── adapters/cache/redis
├── adapters/email/sesv2
├── adapters/queue/rabbitmq
└── … (one entry per adapter)
```

## Dependency Rules

| Layer | May depend on |
|---|---|
| `kernel` | nothing |
| `ports` | `kernel` |
| `app` | `kernel`, `ports` |
| `adapters` | `kernel`, `ports` |
| `testkit` | `kernel`, `ports`, `app` |

Circular dependencies between adapters or between `app` and `adapters` are forbidden.

## Adding a New Adapter

1. Define or reuse a port in `ports/<capability>/port.go`.
2. Create `adapters/<capability>/<provider>/` with its own `go.mod`.
3. Implement the port interface; add `var _ port.Interface = (*Client)(nil)` for compile-time check.
4. Add the module to `go.work`.
5. Write tests. Use `httptest.Server` for HTTP-based providers; use `t.Skip` for integration tests that need real infrastructure.

See [Contributing](contributing.md) for conventions.
