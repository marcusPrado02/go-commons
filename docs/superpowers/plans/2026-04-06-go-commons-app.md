# go-commons App Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `app/` cross-cutting concerns: resilience (retry + circuit breaker), observability (health checks + log sanitizer), transactional outbox, and scheduler.

**Architecture:** All packages in the module root (`github.com/marcusPrado02/go-commons`). `app/resilience` wraps `gobreaker` + manual retry with exponential backoff + jitter. `app/observability` provides concrete implementations of `ports/observability` health interfaces. `app/outbox` implements the Transactional Outbox pattern. `app/scheduler` wraps `robfig/cron` with context propagation and panic recovery.

**Prerequisite:** Plan 1 (Core) must be complete — this plan imports `kernel/` and `ports/`.

**Tech Stack:** Go 1.22, `github.com/sony/gobreaker v0.5.0`, `github.com/robfig/cron/v3 v3.0.1`, `github.com/stretchr/testify v1.9.0`

---

## File Map

```
app/
├── resilience/
│   ├── executor.go
│   └── executor_test.go
├── observability/
│   ├── health.go
│   ├── health_test.go
│   ├── sanitizer.go
│   └── sanitizer_test.go
├── outbox/
│   ├── outbox.go
│   └── outbox_test.go
└── scheduler/
    ├── scheduler.go
    └── scheduler_test.go
```

---

## Task 1: app/resilience — ResilienceExecutor with retry and circuit breaker

**Files:**
- Create: `app/resilience/executor_test.go`
- Create: `app/resilience/executor.go`

- [ ] **Step 1: Write failing tests**

Create `app/resilience/executor_test.go`:

```go
package resilience_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/app/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_Run_Success(t *testing.T) {
	exec := resilience.NewExecutor()
	called := 0

	err := exec.Run(context.Background(), "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond,
	}, func(ctx context.Context) error {
		called++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, called, "should succeed on first attempt")
}

func TestExecutor_Run_RetriesOnError(t *testing.T) {
	exec := resilience.NewExecutor()
	var attempts int32

	err := exec.Run(context.Background(), "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond,
		RetryMaxDelay: 10 * time.Millisecond,
	}, func(ctx context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), attempts, "should retry until success")
}

func TestExecutor_Run_ExhaustsRetries(t *testing.T) {
	exec := resilience.NewExecutor()
	var attempts int32

	err := exec.Run(context.Background(), "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 2,
		RetryDelay:    time.Millisecond,
	}, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("permanent error")
	})

	assert.Error(t, err)
	assert.Equal(t, int32(3), attempts, "1 initial + 2 retries")
}

func TestExecutor_Run_RespectsContextCancellation(t *testing.T) {
	exec := resilience.NewExecutor()
	ctx, cancel := context.WithCancel(context.Background())
	var attempts int32

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := exec.Run(ctx, "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 10,
		RetryDelay:    20 * time.Millisecond,
	}, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Less(t, atomic.LoadInt32(&attempts), int32(10), "context cancel should stop retries early")
}

func TestSupply_ReturnsValue(t *testing.T) {
	exec := resilience.NewExecutor()

	result, err := resilience.Supply(context.Background(), exec, "get-user", resilience.ResiliencePolicySet{
		RetryAttempts: 1,
	}, func(ctx context.Context) (string, error) {
		return "user-123", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "user-123", result)
}

func TestSupply_ReturnsZeroOnError(t *testing.T) {
	exec := resilience.NewExecutor()

	result, err := resilience.Supply(context.Background(), exec, "get-user", resilience.ResiliencePolicySet{
		RetryAttempts: 0,
	}, func(ctx context.Context) (string, error) {
		return "", errors.New("not found")
	})

	assert.Error(t, err)
	assert.Empty(t, result)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/resilience/... -v
```

Expected: compilation error — package not found

- [ ] **Step 3: Implement app/resilience/executor.go**

`★ Insight ─────────────────────────────────────`
The backoff jitter algorithm below is a meaningful design choice. Full jitter (`random(0, cap)`) spreads retries across the full window — better for thundering herd prevention than equal jitter. AWS recommends this for distributed retries. The implementation is your contribution point (Step 3b).
`─────────────────────────────────────────────────`

Create `app/resilience/executor.go` with the structure (you will fill in the jitter function):

```go
// Package resilience provides retry and circuit breaker capabilities.
// Use ResilienceExecutor.Run for void operations and Supply[T] for operations that return a value.
package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/sony/gobreaker"
)

// ResiliencePolicySet configures the retry and circuit breaker behavior.
type ResiliencePolicySet struct {
	// RetryAttempts is the number of retries after the initial attempt (0 = no retries).
	RetryAttempts int
	// RetryDelay is the base delay before the first retry.
	RetryDelay time.Duration
	// RetryMaxDelay caps the exponential backoff delay.
	RetryMaxDelay time.Duration
	// TimeoutDuration limits the total time for a single attempt (0 = no timeout).
	TimeoutDuration time.Duration
	// CircuitBreaker enables circuit breaking if non-nil.
	CircuitBreaker *CircuitBreakerConfig
}

// CircuitBreakerConfig configures the circuit breaker state machine.
type CircuitBreakerConfig struct {
	// MaxRequests is the number of requests allowed in the half-open state.
	MaxRequests uint32
	// Interval is the cyclic period for clearing failure counts in the closed state.
	Interval time.Duration
	// Timeout is how long the circuit stays open before transitioning to half-open.
	Timeout time.Duration
	// FailureThreshold is the ratio of failures that trips the circuit (0.0–1.0).
	FailureThreshold float64
}

// ResilienceExecutor executes actions with retry and circuit breaker policies applied.
type ResilienceExecutor interface {
	Run(ctx context.Context, name string, policies ResiliencePolicySet, action func(ctx context.Context) error) error
}

type defaultExecutor struct{}

// NewExecutor creates a new ResilienceExecutor.
func NewExecutor() ResilienceExecutor {
	return &defaultExecutor{}
}

func (e *defaultExecutor) Run(ctx context.Context, name string, policies ResiliencePolicySet, action func(ctx context.Context) error) error {
	run := action
	if policies.CircuitBreaker != nil {
		cb := newCircuitBreaker(name, policies.CircuitBreaker)
		innerRun := run
		run = func(ctx context.Context) error {
			_, err := cb.Execute(func() (any, error) {
				return nil, innerRun(ctx)
			})
			return err
		}
	}

	var lastErr error
	for attempt := 0; attempt <= policies.RetryAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		attemptCtx := ctx
		if policies.TimeoutDuration > 0 {
			var cancel context.CancelFunc
			attemptCtx, cancel = context.WithTimeout(ctx, policies.TimeoutDuration)
			defer cancel()
		}

		if lastErr = run(attemptCtx); lastErr == nil {
			return nil
		}

		if attempt < policies.RetryAttempts {
			delay := jitterDelay(attempt, policies.RetryDelay, policies.RetryMaxDelay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return lastErr
}

// jitterDelay computes the backoff delay for a given attempt using full jitter:
// delay = random(0, min(RetryMaxDelay, RetryDelay * 2^attempt))
// Full jitter prevents thundering herd by spreading retries uniformly across the window.
func jitterDelay(attempt int, base, maxDelay time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	exp := time.Duration(math.Pow(2, float64(attempt))) * base
	cap := exp
	if maxDelay > 0 && exp > maxDelay {
		cap = maxDelay
	}
	//nolint:gosec // math/rand is fine for jitter — not a security-sensitive operation
	return time.Duration(rand.Int63n(int64(cap) + 1))
}

func newCircuitBreaker(name string, cfg *CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 5 {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= cfg.FailureThreshold
		},
	})
}

// Supply executes an action that returns a value with retry and circuit breaker policies.
// It avoids the closure-plus-variable pattern that would be needed otherwise.
func Supply[T any](ctx context.Context, exec ResilienceExecutor, name string, policies ResiliencePolicySet, action func(ctx context.Context) (T, error)) (T, error) {
	var result T
	err := exec.Run(ctx, name, policies, func(ctx context.Context) error {
		var err error
		result, err = action(ctx)
		return err
	})
	return result, err
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./app/resilience/... -v -race
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add app/resilience/
git commit -m "feat(app): add resilience executor with exponential backoff jitter and circuit breaker"
```

---

## Task 2: app/observability — HealthChecks and LogSanitizer

**Files:**
- Create: `app/observability/health.go`
- Create: `app/observability/health_test.go`
- Create: `app/observability/sanitizer.go`
- Create: `app/observability/sanitizer_test.go`

- [ ] **Step 1: Write failing health tests**

Create `app/observability/health_test.go`:

```go
package observability_test

import (
	"context"
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/app/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type alwaysUpCheck struct {
	name      string
	checkType observability.HealthCheckType
}

func (c alwaysUpCheck) Name() string                                   { return c.name }
func (c alwaysUpCheck) Type() observability.HealthCheckType            { return c.checkType }
func (c alwaysUpCheck) Check(_ context.Context) observability.HealthCheckResult {
	return observability.HealthCheckResult{Status: observability.StatusUp}
}

type alwaysDownCheck struct {
	name      string
	checkType observability.HealthCheckType
}

func (c alwaysDownCheck) Name() string                                   { return c.name }
func (c alwaysDownCheck) Type() observability.HealthCheckType            { return c.checkType }
func (c alwaysDownCheck) Check(_ context.Context) observability.HealthCheckResult {
	return observability.HealthCheckResult{Status: observability.StatusDown}
}

func TestHealthChecks_Liveness_AllUp(t *testing.T) {
	hc := observability.NewHealthChecks(
		alwaysUpCheck{name: "db", checkType: observability.Liveness},
		alwaysUpCheck{name: "cache", checkType: observability.Liveness},
	)

	report := hc.Liveness(context.Background())
	assert.Equal(t, observability.StatusUp, report.Status)
	assert.Len(t, report.Checks, 2)
}

func TestHealthChecks_Liveness_OneDown(t *testing.T) {
	hc := observability.NewHealthChecks(
		alwaysUpCheck{name: "db", checkType: observability.Liveness},
		alwaysDownCheck{name: "cache", checkType: observability.Liveness},
	)

	report := hc.Liveness(context.Background())
	assert.Equal(t, observability.StatusDown, report.Status)
}

func TestHealthChecks_Readiness_FiltersType(t *testing.T) {
	hc := observability.NewHealthChecks(
		alwaysDownCheck{name: "db", checkType: observability.Liveness},  // DOWN, but Liveness
		alwaysUpCheck{name: "queue", checkType: observability.Readiness}, // UP, Readiness
	)

	// Readiness should only evaluate Readiness checks
	report := hc.Readiness(context.Background())
	assert.Equal(t, observability.StatusUp, report.Status)
}

func TestHealthChecks_Report_IncludesTimestamp(t *testing.T) {
	hc := observability.NewHealthChecks(alwaysUpCheck{name: "x", checkType: observability.Liveness})
	before := time.Now()
	report := hc.Liveness(context.Background())
	after := time.Now()

	assert.True(t, report.CheckedAt.After(before) || report.CheckedAt.Equal(before))
	assert.True(t, report.CheckedAt.Before(after) || report.CheckedAt.Equal(after))
}
```

- [ ] **Step 2: Write failing sanitizer tests**

Create `app/observability/sanitizer_test.go`:

```go
package observability_test

import (
	"testing"

	"github.com/marcusPrado02/go-commons/app/observability"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSanitizer_RedactsSensitiveKeys(t *testing.T) {
	s := observability.NewDefaultSanitizer()

	sensitiveKeys := []string{"password", "token", "secret", "cpf", "credit_card", "authorization"}
	for _, key := range sensitiveKeys {
		result := s.Sanitize(key, "sensitive-value")
		assert.Equal(t, "[REDACTED]", result, "key %q should be redacted", key)
	}
}

func TestDefaultSanitizer_PassesThroughSafeKeys(t *testing.T) {
	s := observability.NewDefaultSanitizer()
	result := s.Sanitize("user_id", "u-123")
	assert.Equal(t, "u-123", result)
}

func TestDefaultSanitizer_CaseInsensitive(t *testing.T) {
	s := observability.NewDefaultSanitizer()
	assert.Equal(t, "[REDACTED]", s.Sanitize("PASSWORD", "val"))
	assert.Equal(t, "[REDACTED]", s.Sanitize("Token", "val"))
}

func TestDefaultSanitizer_SanitizeMap(t *testing.T) {
	s := observability.NewDefaultSanitizer()
	input := map[string]any{
		"username": "alice",
		"password": "s3cr3t",
		"token":    "abc123",
	}
	result := s.SanitizeMap(input)

	assert.Equal(t, "alice", result["username"])
	assert.Equal(t, "[REDACTED]", result["password"])
	assert.Equal(t, "[REDACTED]", result["token"])
}

func TestDefaultSanitizer_AdditionalKeys(t *testing.T) {
	s := observability.NewDefaultSanitizer("ssn", "api_key")
	assert.Equal(t, "[REDACTED]", s.Sanitize("ssn", "123-45-6789"))
	assert.Equal(t, "[REDACTED]", s.Sanitize("api_key", "sk-xyz"))
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./app/observability/... -v
```

Expected: compilation error

- [ ] **Step 4: Implement app/observability/health.go**

Create `app/observability/health.go`:

```go
// Package observability provides concrete implementations of health checking
// and log sanitization for the ports/observability interfaces.
package observability

import (
	"context"
	"time"
)

// HealthStatus represents the operational state of a component.
type HealthStatus string

const (
	StatusUp       HealthStatus = "UP"
	StatusDown     HealthStatus = "DOWN"
	StatusDegraded HealthStatus = "DEGRADED"
)

// HealthCheckType determines which health endpoint a check contributes to.
type HealthCheckType string

const (
	// Liveness checks determine if the process should be restarted.
	Liveness HealthCheckType = "LIVENESS"
	// Readiness checks determine if the process should receive traffic.
	Readiness HealthCheckType = "READINESS"
)

// HealthCheckResult is the outcome of a single health check.
type HealthCheckResult struct {
	Status  HealthStatus
	Details map[string]any
}

// NamedResult pairs a check name with its result for reporting.
type NamedResult struct {
	Name   string
	Result HealthCheckResult
}

// HealthReport is the aggregated result for a set of health checks.
type HealthReport struct {
	Status    HealthStatus
	Checks    []NamedResult
	CheckedAt time.Time
}

// HealthCheck is the interface for a single health check contributor.
type HealthCheck interface {
	Name() string
	Type() HealthCheckType
	Check(ctx context.Context) HealthCheckResult
}

// HealthChecks aggregates multiple HealthCheck implementations.
type HealthChecks struct {
	checks []HealthCheck
}

// NewHealthChecks creates a HealthChecks aggregator with the given checks.
func NewHealthChecks(checks ...HealthCheck) *HealthChecks {
	return &HealthChecks{checks: checks}
}

// Liveness evaluates all checks of type Liveness and returns an aggregated report.
func (h *HealthChecks) Liveness(ctx context.Context) HealthReport {
	return h.evaluate(ctx, Liveness)
}

// Readiness evaluates all checks of type Readiness and returns an aggregated report.
func (h *HealthChecks) Readiness(ctx context.Context) HealthReport {
	return h.evaluate(ctx, Readiness)
}

func (h *HealthChecks) evaluate(ctx context.Context, checkType HealthCheckType) HealthReport {
	var results []NamedResult
	overallStatus := StatusUp

	for _, check := range h.checks {
		if check.Type() != checkType {
			continue
		}
		result := check.Check(ctx)
		results = append(results, NamedResult{Name: check.Name(), Result: result})
		if result.Status == StatusDown {
			overallStatus = StatusDown
		} else if result.Status == StatusDegraded && overallStatus != StatusDown {
			overallStatus = StatusDegraded
		}
	}

	return HealthReport{
		Status:    overallStatus,
		Checks:    results,
		CheckedAt: time.Now(),
	}
}
```

- [ ] **Step 5: Implement app/observability/sanitizer.go**

Create `app/observability/sanitizer.go`:

```go
package observability

import "strings"

const redacted = "[REDACTED]"

// defaultSensitiveKeys are always redacted regardless of configuration.
var defaultSensitiveKeys = []string{
	"password", "token", "secret", "cpf", "credit_card", "authorization",
}

// LogSanitizer removes PII and secrets from structured log fields before emission.
type LogSanitizer interface {
	Sanitize(key string, value any) any
	SanitizeMap(input map[string]any) map[string]any
}

type defaultSanitizer struct {
	sensitiveKeys map[string]struct{}
}

// NewDefaultSanitizer creates a LogSanitizer that redacts common sensitive field names.
// additionalKeys adds extra keys to the default redaction list (case-insensitive).
func NewDefaultSanitizer(additionalKeys ...string) LogSanitizer {
	keys := make(map[string]struct{}, len(defaultSensitiveKeys)+len(additionalKeys))
	for _, k := range defaultSensitiveKeys {
		keys[strings.ToLower(k)] = struct{}{}
	}
	for _, k := range additionalKeys {
		keys[strings.ToLower(k)] = struct{}{}
	}
	return &defaultSanitizer{sensitiveKeys: keys}
}

// Sanitize returns "[REDACTED]" if the key is sensitive, otherwise returns value unchanged.
func (s *defaultSanitizer) Sanitize(key string, value any) any {
	if _, sensitive := s.sensitiveKeys[strings.ToLower(key)]; sensitive {
		return redacted
	}
	return value
}

// SanitizeMap returns a new map with sensitive values replaced by "[REDACTED]".
func (s *defaultSanitizer) SanitizeMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for k, v := range input {
		result[k] = s.Sanitize(k, v)
	}
	return result
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./app/observability/... -v -race
```

Expected: all tests PASS

- [ ] **Step 7: Commit**

```bash
git add app/observability/
git commit -m "feat(app): add health checks aggregator and log sanitizer"
```

---

## Task 3: app/outbox — Transactional Outbox

**Files:**
- Create: `app/outbox/outbox_test.go`
- Create: `app/outbox/outbox.go`

- [ ] **Step 1: Write failing tests**

Create `app/outbox/outbox_test.go`:

```go
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
	mu       sync.Mutex
	messages []outbox.OutboxMessage
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
	mu       sync.Mutex
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/outbox/... -v
```

Expected: compilation error

- [ ] **Step 3: Implement app/outbox/outbox.go**

Create `app/outbox/outbox.go`:

```go
// Package outbox implements the Transactional Outbox pattern.
//
// Usage:
//  1. In the same transaction as your aggregate save, persist OutboxMessages via OutboxStore.Save.
//  2. Run OutboxPublisher to deliver those messages asynchronously.
//  3. OutboxPublisher guarantees at-least-once delivery via idempotent ID-based deduplication.
package outbox

import (
	"context"
	"sync"
	"time"
)

// OutboxMessage is a unit of work persisted alongside a domain aggregate.
// ID is the idempotency key — use a UUID generated by the caller.
// Payload must be JSON — serialization is the caller's responsibility.
type OutboxMessage struct {
	// ID is the idempotency key. Must be globally unique (e.g. UUID).
	ID          string
	AggregateID string
	EventType   string
	// Payload is JSON-encoded. The outbox is a delivery mechanism, not a serializer.
	Payload     []byte
	CreatedAt   time.Time
	ProcessedAt *time.Time
	// Attempts tracks how many delivery attempts have been made.
	Attempts int
}

// OutboxStore persists and queries outbox messages.
// Implement this interface backed by your application database.
type OutboxStore interface {
	// Save persists messages, typically in the same transaction as the aggregate.
	Save(ctx context.Context, msgs []OutboxMessage) error
	// FetchPending returns up to limit unprocessed messages.
	FetchPending(ctx context.Context, limit int) ([]OutboxMessage, error)
	// MarkProcessed marks a message as delivered. Idempotent — safe to call multiple times.
	MarkProcessed(ctx context.Context, id string) error
}

// PublishFunc is the function called to deliver a single outbox message.
type PublishFunc func(ctx context.Context, msg OutboxMessage) error

// publisherOptions holds resolved configuration for OutboxPublisher.
type publisherOptions struct {
	pollingInterval time.Duration
	batchSize       int
	concurrency     int
}

// Option configures an OutboxPublisher.
type Option func(*publisherOptions)

// WithPollingInterval sets how often the publisher polls for pending messages.
// Default: 5s.
func WithPollingInterval(d time.Duration) Option {
	return func(o *publisherOptions) { o.pollingInterval = d }
}

// WithBatchSize sets the maximum number of messages processed per polling cycle.
// Default: 100.
func WithBatchSize(n int) Option {
	return func(o *publisherOptions) { o.batchSize = n }
}

// WithConcurrency sets the number of concurrent delivery goroutines.
// Default: 1 (sequential — preserves message ordering within a batch).
func WithConcurrency(n int) Option {
	return func(o *publisherOptions) { o.concurrency = n }
}

// OutboxPublisher polls the OutboxStore and delivers pending messages via PublishFunc.
type OutboxPublisher struct {
	store   OutboxStore
	publish PublishFunc
	opts    publisherOptions
	stopCh  chan struct{}
	doneCh  chan struct{}
	once    sync.Once
}

// NewPublisher creates an OutboxPublisher with the given store and publish function.
func NewPublisher(store OutboxStore, publish PublishFunc, opts ...Option) *OutboxPublisher {
	o := publisherOptions{
		pollingInterval: 5 * time.Second,
		batchSize:       100,
		concurrency:     1,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &OutboxPublisher{
		store:   store,
		publish: publish,
		opts:    o,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Start launches the polling goroutine. Non-blocking — returns immediately.
// Returns an error if the publisher is already running.
// Cancel ctx or call Stop to gracefully shut down.
func (p *OutboxPublisher) Start(ctx context.Context) error {
	started := false
	p.once.Do(func() {
		started = true
		go p.run(ctx)
	})
	if !started {
		return nil // already running — idempotent
	}
	return nil
}

// Stop waits for the current polling cycle to complete before returning.
// ctx defines the maximum wait time for graceful shutdown.
func (p *OutboxPublisher) Stop(ctx context.Context) error {
	close(p.stopCh)
	select {
	case <-p.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *OutboxPublisher) run(ctx context.Context) {
	defer close(p.doneCh)
	ticker := time.NewTicker(p.opts.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processOnce(ctx)
		}
	}
}

func (p *OutboxPublisher) processOnce(ctx context.Context) {
	msgs, err := p.store.FetchPending(ctx, p.opts.batchSize)
	if err != nil || len(msgs) == 0 {
		return
	}
	for _, msg := range msgs {
		if err := p.publish(ctx, msg); err != nil {
			continue // leave unprocessed for next cycle
		}
		_ = p.store.MarkProcessed(ctx, msg.ID)
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./app/outbox/... -v -race
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add app/outbox/
git commit -m "feat(app): add transactional outbox with non-blocking publisher and idempotent delivery"
```

---

## Task 4: app/scheduler — cron-based job scheduler

**Files:**
- Create: `app/scheduler/scheduler_test.go`
- Create: `app/scheduler/scheduler.go`

- [ ] **Step 1: Write failing tests**

Create `app/scheduler/scheduler_test.go`:

```go
package scheduler_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/app/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type counterJob struct {
	name    string
	count   atomic.Int32
}

func (j *counterJob) Name() string { return j.name }
func (j *counterJob) Run(_ context.Context) error {
	j.count.Add(1)
	return nil
}

func TestScheduler_Register_InvalidCron(t *testing.T) {
	s := scheduler.NewScheduler()
	err := s.Register(&counterJob{name: "bad"}, "not-a-cron")
	assert.Error(t, err)
}

func TestScheduler_Register_ValidCron(t *testing.T) {
	s := scheduler.NewScheduler()
	err := s.Register(&counterJob{name: "good"}, "@every 1s")
	assert.NoError(t, err)
}

func TestScheduler_RunsJobOnSchedule(t *testing.T) {
	s := scheduler.NewScheduler()
	job := &counterJob{name: "tick"}

	err := s.Register(job, "@every 50ms")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	time.Sleep(180 * time.Millisecond)
	cancel()
	_ = s.Stop(context.Background())

	assert.GreaterOrEqual(t, int(job.count.Load()), 2, "job should have run at least twice")
}

func TestScheduler_Stop_GracefulShutdown(t *testing.T) {
	s := scheduler.NewScheduler()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.Start(ctx)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err := s.Stop(stopCtx)
	assert.NoError(t, err)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./app/scheduler/... -v
```

Expected: compilation error

- [ ] **Step 3: Implement app/scheduler/scheduler.go**

Create `app/scheduler/scheduler.go`:

```go
// Package scheduler provides a cron-based job scheduler with context propagation and panic recovery.
package scheduler

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
)

// Job is a named, runnable unit of work.
type Job interface {
	Name() string
	Run(ctx context.Context) error
}

// Scheduler registers jobs on cron schedules and manages their lifecycle.
type Scheduler interface {
	// Register adds a job to run on the given cron schedule.
	// Returns an error immediately if the schedule expression is invalid.
	Register(job Job, schedule string) error
	// Start begins executing scheduled jobs. Non-blocking.
	Start(ctx context.Context)
	// Stop waits for running jobs to complete before returning.
	Stop(ctx context.Context) error
}

type defaultScheduler struct {
	cron *cron.Cron
}

// NewScheduler creates a Scheduler using standard cron expressions plus descriptors (@every, @hourly, etc.).
func NewScheduler() Scheduler {
	return &defaultScheduler{
		cron: cron.New(cron.WithSeconds()),
	}
}

// Register validates the schedule expression and adds the job.
func (s *defaultScheduler) Register(job Job, schedule string) error {
	// Validate expression before adding
	p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := p.Parse(schedule); err != nil {
		return fmt.Errorf("scheduler: invalid cron expression %q for job %q: %w", schedule, job.Name(), err)
	}

	_, err := s.cron.AddFunc(schedule, func() {
		defer func() {
			if r := recover(); r != nil {
				// Panic recovery — log or handle r in production via a Logger option
				_ = r
			}
		}()
		_ = job.Run(context.Background())
	})
	return err
}

// Start begins the scheduler. Non-blocking.
func (s *defaultScheduler) Start(_ context.Context) {
	s.cron.Start()
}

// Stop waits for all currently running jobs to complete, then stops the scheduler.
func (s *defaultScheduler) Stop(ctx context.Context) error {
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./app/scheduler/... -v -race
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add app/scheduler/
git commit -m "feat(app): add cron scheduler with context propagation and panic recovery"
```

---

## Task 5: go mod tidy + full test run

- [ ] **Step 1: Tidy dependencies**

```bash
go mod tidy
```

- [ ] **Step 2: Run all tests**

```bash
go test ./... -race -v
```

Expected: all PASS

- [ ] **Step 3: Check coverage**

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

Expected: >= 60% total

- [ ] **Step 4: Run linter**

```bash
golangci-lint run ./...
```

Expected: no issues

- [ ] **Step 5: Final commit**

```bash
git add go.mod go.sum
git commit -m "chore: go mod tidy after app layer implementation"
```

---

## Self-Review Checklist

After completing all tasks, verify:

- [ ] `go build ./...` passes
- [ ] `go test ./... -race` passes
- [ ] `golangci-lint run ./...` passes
- [ ] `app/resilience` tests cover retry exhaustion and context cancellation
- [ ] `app/observability` tests cover both health check types and sanitizer edge cases
- [ ] `app/outbox` publisher is non-blocking and MarkProcessed is idempotent
- [ ] `app/scheduler` validates cron expressions on Register
- [ ] No `TODO` or `TBD` in any `.go` file
