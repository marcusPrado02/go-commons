# go-commons Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the pure-Go core of go-commons: module foundation, kernel primitives (errors, result, ddd), all port interfaces, and the testkit.

**Architecture:** Single Go module (`github.com/marcusPrado02/go-commons`) containing `kernel/`, `ports/`, `app/` (app layer in Plan 2), and `testkit/`. Zero external deps in `kernel/` and `ports/`. `testkit/` depends on `testify`. All code test-driven.

**Tech Stack:** Go 1.22, `github.com/stretchr/testify v1.9.0`

---

## File Map

```
go-commons/
├── go.mod
├── go.work                                  ← updated with adapter submódulos as they are added
├── Makefile
├── .golangci.yml
├── kernel/
│   ├── errors/
│   │   ├── errors.go
│   │   └── errors_test.go
│   ├── result/
│   │   ├── result.go
│   │   └── result_test.go
│   └── ddd/
│       ├── aggregate.go
│       └── aggregate_test.go
├── ports/
│   ├── email/
│   │   └── port.go
│   ├── files/
│   │   └── port.go
│   ├── persistence/
│   │   └── repository.go
│   ├── template/
│   │   └── port.go
│   ├── observability/
│   │   ├── field.go
│   │   ├── interfaces.go
│   │   └── conventions.go
│   ├── cache/
│   │   └── port.go
│   ├── queue/
│   │   └── port.go
│   ├── sms/
│   │   └── port.go
│   ├── push/
│   │   └── port.go
│   ├── secrets/
│   │   └── port.go
│   ├── excel/
│   │   └── port.go
│   └── compression/
│       └── port.go
└── testkit/
    ├── assert/
    │   └── aggregate.go
    └── contracts/
        └── repository.go
```

---

## Task 1: Foundation — go.mod, go.work, Makefile, .golangci.yml

**Files:**
- Create: `go.mod`
- Create: `go.work`
- Create: `Makefile`
- Create: `.golangci.yml`

- [ ] **Step 1: Create go.mod**

```
module github.com/marcusPrado02/go-commons

go 1.22.0

require (
	github.com/robfig/cron/v3 v3.0.1
	github.com/sony/gobreaker v0.5.0
	github.com/stretchr/testify v1.9.0
)
```

- [ ] **Step 2: Create go.work**

```
go 1.22.0

use (
	.
)
```

> `go.work` will grow as adapter submódulos are added in Plans 2 and 3.

- [ ] **Step 3: Create Makefile**

```makefile
.PHONY: build test lint coverage tidy tidy-all

build:
	go build ./...

test:
	go test ./... -race

lint:
	golangci-lint run ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total

tidy:
	go mod tidy

tidy-all:
	go mod tidy
	@for d in $$(find adapters -name "go.mod" -exec dirname {} \;); do \
		echo "Tidying $$d"; \
		(cd $$d && go mod tidy); \
	done
```

- [ ] **Step 4: Create .golangci.yml**

```yaml
linters:
  enable:
    - errcheck
    - staticcheck
    - revive
    - govet
    - gosimple
    - unused
    - misspell

linters-settings:
  revive:
    rules:
      - name: exported
        arguments: ["checkPrivateReceivers", "sayRepetitiveInsteadOfStutters"]

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

- [ ] **Step 5: Run go mod tidy and verify**

```bash
go mod tidy
go build ./...
```

Expected: no errors (nothing to build yet, but module is valid)

- [ ] **Step 6: Commit**

```bash
git add go.mod go.work Makefile .golangci.yml
git commit -m "chore: initialize go module with workspace, Makefile and linter config"
```

---

## Task 2: kernel/errors — types and Problem

**Files:**
- Create: `kernel/errors/errors_test.go`
- Create: `kernel/errors/errors.go`

- [ ] **Step 1: Write failing tests**

Create `kernel/errors/errors_test.go`:

```go
package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/marcusPrado02/go-commons/kernel/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrorCode_valid(t *testing.T) {
	code, err := errors.NewErrorCode("USER_NOT_FOUND")
	require.NoError(t, err)
	assert.Equal(t, errors.ErrorCode("USER_NOT_FOUND"), code)
}

func TestNewErrorCode_empty(t *testing.T) {
	_, err := errors.NewErrorCode("")
	assert.Error(t, err)
}

func TestProblem_Error(t *testing.T) {
	p := errors.NewProblem("TEST_CODE", errors.CategoryBusiness, errors.SeverityError, "test message")
	assert.Equal(t, "[TEST_CODE] test message", p.Error())
}

func TestProblem_ErrorWithCause(t *testing.T) {
	cause := stderrors.New("underlying error")
	p := errors.NewProblem("TEST_CODE", errors.CategoryTechnical, errors.SeverityError, "wrapped").
		WithCause(cause)
	assert.Contains(t, p.Error(), "underlying error")
	assert.Equal(t, cause, stderrors.Unwrap(p))
}

func TestProblem_WithDetail(t *testing.T) {
	p := errors.NewProblem("CODE", errors.CategoryValidation, errors.SeverityWarning, "msg")
	p2 := p.WithDetail("field", "email")

	// original is unchanged
	assert.Empty(t, p.Details)
	// copy has the detail
	assert.Equal(t, "email", p2.Details["field"])
}

func TestProblem_WithDetails_merges(t *testing.T) {
	p := errors.NewProblem("CODE", errors.CategoryValidation, errors.SeverityWarning, "msg").
		WithDetail("a", 1)
	p2 := p.WithDetails(map[string]any{"b": 2})

	assert.Equal(t, 1, p2.Details["a"])
	assert.Equal(t, 2, p2.Details["b"])
	assert.NotContains(t, p.Details, "b") // original unchanged
}

func TestProblem_ImplementsError(t *testing.T) {
	var err error = errors.NewProblem("CODE", errors.CategoryBusiness, errors.SeverityError, "msg")
	assert.NotNil(t, err)
}

func TestSentinelErrors_defined(t *testing.T) {
	assert.NotEmpty(t, errors.ErrNotFound.Code)
	assert.NotEmpty(t, errors.ErrUnauthorized.Code)
	assert.NotEmpty(t, errors.ErrValidation.Code)
	assert.NotEmpty(t, errors.ErrTechnical.Code)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./kernel/errors/... -v
```

Expected: compilation error — `package errors` does not exist yet

- [ ] **Step 3: Implement kernel/errors/errors.go**

Create `kernel/errors/errors.go`:

```go
// Package errors provides domain error types for go-commons.
// It defines Problem, DomainError, and pre-defined sentinel errors.
package errors

import "fmt"

// ErrorCode identifies a specific domain error condition.
type ErrorCode string

// NewErrorCode validates and creates an ErrorCode.
func NewErrorCode(code string) (ErrorCode, error) {
	if code == "" {
		return "", fmt.Errorf("error code cannot be empty")
	}
	return ErrorCode(code), nil
}

// ErrorCategory classifies the nature of a domain error.
type ErrorCategory string

const (
	CategoryValidation   ErrorCategory = "VALIDATION"
	CategoryBusiness     ErrorCategory = "BUSINESS"
	CategoryTechnical    ErrorCategory = "TECHNICAL"
	CategoryNotFound     ErrorCategory = "NOT_FOUND"
	CategoryUnauthorized ErrorCategory = "UNAUTHORIZED"
)

// Severity indicates how critical a domain error is.
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityError    Severity = "ERROR"
	SeverityCritical Severity = "CRITICAL"
)

// Problem is an immutable, rich domain error. Use the With* builders to add context.
// All With* methods return a new copy — the receiver is never modified.
type Problem struct {
	Code     ErrorCode
	Category ErrorCategory
	Severity Severity
	Message  string
	// Details is a defensive copy — safe to read, not to mutate.
	Details map[string]any
	// Cause is the underlying error, preserved for logging and errors.Is/As chaining.
	Cause error
}

// NewProblem creates a Problem with an empty Details map.
func NewProblem(code ErrorCode, category ErrorCategory, severity Severity, message string) Problem {
	return Problem{
		Code:     code,
		Category: category,
		Severity: severity,
		Message:  message,
		Details:  make(map[string]any),
	}
}

// Error implements the error interface.
func (p Problem) Error() string {
	if p.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", p.Code, p.Message, p.Cause)
	}
	return fmt.Sprintf("[%s] %s", p.Code, p.Message)
}

// Unwrap returns the cause, enabling errors.Is/As chaining.
func (p Problem) Unwrap() error { return p.Cause }

// WithDetail returns a new Problem with the key-value pair added to Details.
func (p Problem) WithDetail(key string, value any) Problem {
	details := make(map[string]any, len(p.Details)+1)
	for k, v := range p.Details {
		details[k] = v
	}
	details[key] = value
	p.Details = details
	return p
}

// WithDetails returns a new Problem with the given map merged into Details.
func (p Problem) WithDetails(extra map[string]any) Problem {
	merged := make(map[string]any, len(p.Details)+len(extra))
	for k, v := range p.Details {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	p.Details = merged
	return p
}

// WithCause returns a new Problem with the given cause attached.
func (p Problem) WithCause(err error) Problem {
	p.Cause = err
	return p
}

// DomainError is the interface implemented by errors returned from ports.
// Adapters wrap SDK errors into DomainError before returning.
type DomainError interface {
	error
	Code() ErrorCode
	Category() ErrorCategory
	Severity() Severity
	Details() map[string]any
	Unwrap() error
}

// Pre-defined sentinel errors for common domain conditions.
var (
	ErrNotFound     = NewProblem("NOT_FOUND", CategoryNotFound, SeverityError, "resource not found")
	ErrUnauthorized = NewProblem("UNAUTHORIZED", CategoryUnauthorized, SeverityWarning, "unauthorized access")
	ErrValidation   = NewProblem("VALIDATION", CategoryValidation, SeverityWarning, "validation failed")
	ErrTechnical    = NewProblem("TECHNICAL", CategoryTechnical, SeverityError, "technical error")
)
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./kernel/errors/... -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add kernel/errors/
git commit -m "feat(kernel): add errors package with Problem, ErrorCode, DomainError"
```

---

## Task 3: kernel/result — generic Result type

**Files:**
- Create: `kernel/result/result_test.go`
- Create: `kernel/result/result.go`

- [ ] **Step 1: Write failing tests**

Create `kernel/result/result_test.go`:

```go
package result_test

import (
	"errors"
	"testing"

	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
	"github.com/marcusPrado02/go-commons/kernel/result"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOk_IsOk(t *testing.T) {
	r := result.Ok(42)
	assert.True(t, r.IsOk())
	assert.False(t, r.IsFail())
	assert.Equal(t, 42, r.Value())
}

func TestFail_IsFail(t *testing.T) {
	p := kerrors.NewProblem("ERR", kerrors.CategoryBusiness, kerrors.SeverityError, "bad")
	r := result.Fail[int](p)
	assert.True(t, r.IsFail())
	assert.False(t, r.IsOk())
	assert.Equal(t, p, r.Problem())
}

func TestValue_PanicsOnFail(t *testing.T) {
	p := kerrors.ErrNotFound
	r := result.Fail[string](p)
	assert.Panics(t, func() { r.Value() })
}

func TestProblem_PanicsOnOk(t *testing.T) {
	r := result.Ok("hello")
	assert.Panics(t, func() { r.Problem() })
}

func TestValueOrZero_ReturnsZeroOnFail(t *testing.T) {
	r := result.Fail[int](kerrors.ErrTechnical)
	assert.Equal(t, 0, r.ValueOrZero())
}

func TestUnwrap_Success(t *testing.T) {
	r := result.Ok("val")
	v, err := r.Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "val", v)
}

func TestUnwrap_Failure(t *testing.T) {
	r := result.Fail[string](kerrors.ErrNotFound)
	v, err := r.Unwrap()
	assert.Error(t, err)
	assert.Empty(t, v)
}

func TestFromError_WithNilError(t *testing.T) {
	r := result.FromError("hello", nil)
	assert.True(t, r.IsOk())
	assert.Equal(t, "hello", r.Value())
}

func TestFromError_WithError(t *testing.T) {
	r := result.FromError("", errors.New("something went wrong"))
	assert.True(t, r.IsFail())
	assert.Equal(t, kerrors.CategoryTechnical, r.Problem().Category)
}

func TestFromError_WithProblem(t *testing.T) {
	p := kerrors.ErrNotFound
	r := result.FromError("", p)
	assert.True(t, r.IsFail())
	assert.Equal(t, kerrors.CategoryNotFound, r.Problem().Category)
}

func TestMap_TransformsValue(t *testing.T) {
	r := result.Ok(2)
	doubled := result.Map(r, func(n int) string { return fmt.Sprintf("%dx", n) })
	assert.True(t, doubled.IsOk())
	assert.Equal(t, "2x", doubled.Value())
}

func TestMap_PropagatesFail(t *testing.T) {
	r := result.Fail[int](kerrors.ErrNotFound)
	mapped := result.Map(r, func(n int) string { return "should not run" })
	assert.True(t, mapped.IsFail())
}

func TestFlatMap_ChainsSuccess(t *testing.T) {
	r := result.Ok(5)
	chained := result.FlatMap(r, func(n int) result.Result[string] {
		if n > 3 {
			return result.Ok("big")
		}
		return result.Fail[string](kerrors.ErrValidation)
	})
	assert.True(t, chained.IsOk())
	assert.Equal(t, "big", chained.Value())
}

func TestFlatMap_PropagatesFail(t *testing.T) {
	r := result.Fail[int](kerrors.ErrUnauthorized)
	chained := result.FlatMap(r, func(n int) result.Result[string] { return result.Ok("x") })
	assert.True(t, chained.IsFail())
	assert.Equal(t, kerrors.CategoryUnauthorized, chained.Problem().Category)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./kernel/result/... -v
```

Expected: compilation error — package not found

- [ ] **Step 3: Add missing import to test file**

Add `"fmt"` to the imports in `result_test.go`.

- [ ] **Step 4: Implement kernel/result/result.go**

Create `kernel/result/result.go`:

```go
// Package result provides a generic Result[T] type for functional pipelines.
// Port interfaces use (T, error) — Result[T] is an opt-in utility for
// cases where chaining transformations is more expressive than sequential if-err checks.
package result

import (
	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
)

// Result represents a computation that either succeeded with a value or failed with a Problem.
type Result[T any] struct {
	value   T
	problem *kerrors.Problem
}

// Ok creates a successful Result holding the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Fail creates a failed Result holding the given Problem.
func Fail[T any](problem kerrors.Problem) Result[T] {
	return Result[T]{problem: &problem}
}

// FromError bridges idiomatic Go (T, error) into Result[T].
// If err is nil, returns Ok(value). If err is a Problem, wraps it directly.
// Otherwise wraps it in ErrTechnical.
func FromError[T any](value T, err error) Result[T] {
	if err == nil {
		return Ok(value)
	}
	if prob, ok := err.(kerrors.Problem); ok {
		return Fail[T](prob)
	}
	return Fail[T](kerrors.ErrTechnical.WithCause(err))
}

// IsOk returns true if the Result holds a value.
func (r Result[T]) IsOk() bool { return r.problem == nil }

// IsFail returns true if the Result holds a Problem.
func (r Result[T]) IsFail() bool { return r.problem != nil }

// Value returns the held value. Panics if IsFail() — only call when IsOk() is guaranteed.
func (r Result[T]) Value() T {
	if r.IsFail() {
		panic("result: called Value() on a failed Result — check IsOk() first")
	}
	return r.value
}

// ValueOrZero returns the held value, or the zero value of T if IsFail().
func (r Result[T]) ValueOrZero() T { return r.value }

// Problem returns the held Problem. Panics if IsOk() — only call when IsFail() is guaranteed.
func (r Result[T]) Problem() kerrors.Problem {
	if r.IsOk() {
		panic("result: called Problem() on a successful Result — check IsFail() first")
	}
	return *r.problem
}

// Unwrap returns (value, nil) on success or (zero, problem) on failure.
// Use this when integrating with code that expects idiomatic (T, error).
func (r Result[T]) Unwrap() (T, error) {
	if r.IsFail() {
		var zero T
		return zero, *r.problem
	}
	return r.value, nil
}

// Map transforms a successful Result[T] into Result[U] by applying f.
// If r is failed, the failure propagates unchanged.
func Map[T, U any](r Result[T], f func(T) U) Result[U] {
	if r.IsFail() {
		return Fail[U](r.Problem())
	}
	return Ok(f(r.value))
}

// FlatMap chains a successful Result[T] with a function returning Result[U].
// If r is failed, the failure propagates unchanged and f is never called.
func FlatMap[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	if r.IsFail() {
		return Fail[U](r.Problem())
	}
	return f(r.value)
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./kernel/result/... -v
```

Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add kernel/result/
git commit -m "feat(kernel): add generic Result[T] with Ok, Fail, FromError, Map, FlatMap"
```

---

## Task 4: kernel/ddd — AggregateRoot and DomainEvent

**Files:**
- Create: `kernel/ddd/aggregate_test.go`
- Create: `kernel/ddd/aggregate.go`

- [ ] **Step 1: Write failing tests**

Create `kernel/ddd/aggregate_test.go`:

```go
package ddd_test

import (
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/kernel/ddd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent is a minimal DomainEvent implementation for tests.
type testEvent struct {
	eventType  string
	occurredAt time.Time
}

func (e testEvent) EventType() string    { return e.eventType }
func (e testEvent) OccurredAt() time.Time { return e.occurredAt }

// testAggregate embeds AggregateRoot for testing.
type testAggregate struct {
	ddd.AggregateRoot[string]
}

func TestAggregateRoot_ID(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	assert.Equal(t, "agg-1", agg.ID())
}

func TestAggregateRoot_RegisterAndPull(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}

	evt := testEvent{eventType: "OrderPlaced", occurredAt: time.Now()}
	agg.RegisterEvent(evt)

	events := agg.PullDomainEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "OrderPlaced", events[0].EventType())
}

func TestAggregateRoot_PullClearsEvents(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	agg.RegisterEvent(testEvent{eventType: "Evt", occurredAt: time.Now()})

	agg.PullDomainEvents()
	second := agg.PullDomainEvents()

	assert.Empty(t, second)
}

func TestAggregateRoot_PullReturnsCopy(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	agg.RegisterEvent(testEvent{eventType: "Evt", occurredAt: time.Now()})

	events := agg.PullDomainEvents()
	// Mutate the returned slice — should not affect the aggregate
	events[0] = testEvent{eventType: "Mutated", occurredAt: time.Now()}

	agg.RegisterEvent(testEvent{eventType: "Second", occurredAt: time.Now()})
	second := agg.PullDomainEvents()
	require.Len(t, second, 1)
	assert.Equal(t, "Second", second[0].EventType())
}

func TestAggregateRoot_NoEventsInitially(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	assert.Empty(t, agg.PullDomainEvents())
}

func TestAggregateRoot_MultipleEvents(t *testing.T) {
	agg := testAggregate{AggregateRoot: ddd.NewAggregateRoot("agg-1")}
	agg.RegisterEvent(testEvent{eventType: "A", occurredAt: time.Now()})
	agg.RegisterEvent(testEvent{eventType: "B", occurredAt: time.Now()})
	agg.RegisterEvent(testEvent{eventType: "C", occurredAt: time.Now()})

	events := agg.PullDomainEvents()
	require.Len(t, events, 3)
	assert.Equal(t, "A", events[0].EventType())
	assert.Equal(t, "B", events[1].EventType())
	assert.Equal(t, "C", events[2].EventType())
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./kernel/ddd/... -v
```

Expected: compilation error

- [ ] **Step 3: Implement kernel/ddd/aggregate.go**

Create `kernel/ddd/aggregate.go`:

```go
// Package ddd provides Domain-Driven Design primitives for go-commons.
// AggregateRoot is designed to be embedded — not extended via inheritance.
//
// Example usage:
//
//	type Order struct {
//	    ddd.AggregateRoot[OrderID]
//	    status OrderStatus
//	}
//
//	func PlaceOrder(id OrderID) *Order {
//	    o := &Order{AggregateRoot: ddd.NewAggregateRoot(id)}
//	    o.RegisterEvent(OrderPlaced{OccurredAt: time.Now()})
//	    return o
//	}
package ddd

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	OccurredAt() time.Time
	EventType() string
}

// AggregateRoot holds the aggregate's identity and pending domain events.
// Embed it in your aggregate struct — never inherit from it.
type AggregateRoot[ID any] struct {
	id     ID
	events []DomainEvent
}

// NewAggregateRoot creates an AggregateRoot with the given identifier.
func NewAggregateRoot[ID any](id ID) AggregateRoot[ID] {
	return AggregateRoot[ID]{id: id}
}

// ID returns the aggregate's identifier.
func (a *AggregateRoot[ID]) ID() ID { return a.id }

// RegisterEvent appends a domain event to the aggregate's pending event list.
// Events are not published until PullDomainEvents is called.
func (a *AggregateRoot[ID]) RegisterEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

// PullDomainEvents returns a copy of all pending domain events and clears the list.
// Safe to call multiple times — subsequent calls return empty slices until new events are registered.
func (a *AggregateRoot[ID]) PullDomainEvents() []DomainEvent {
	events := append([]DomainEvent(nil), a.events...)
	a.events = nil
	return events
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./kernel/ddd/... -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add kernel/ddd/
git commit -m "feat(kernel): add DomainEvent interface and generic AggregateRoot[ID]"
```

---

## Task 5: ports/email and ports/files

**Files:**
- Create: `ports/email/port.go`
- Create: `ports/files/port.go`

> Port interfaces have no logic to test directly. Each adapter will have a compile-time interface check `var _ EmailPort = (*MyAdapter)(nil)` in its own package. Here we define the contracts.

- [ ] **Step 1: Create ports/email/port.go**

```go
// Package email defines the port interface for email delivery.
package email

import (
	"context"
	"fmt"
	"net/mail"
)

// EmailPort is the primary port for sending email messages.
type EmailPort interface {
	// Send delivers a single email message.
	Send(ctx context.Context, email Email) (EmailReceipt, error)
	// SendWithTemplate delivers an email using a named template.
	SendWithTemplate(ctx context.Context, req TemplateEmailRequest) (EmailReceipt, error)
	// Ping verifies the email provider is reachable and credentials are valid.
	Ping(ctx context.Context) error
}

// EmailAddress is a validated email address value object.
// Always construct via NewEmailAddress — never create the struct directly.
type EmailAddress struct {
	Value string
}

// NewEmailAddress parses and validates an email address per RFC 5322.
func NewEmailAddress(value string) (EmailAddress, error) {
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return EmailAddress{}, fmt.Errorf("invalid email address %q: %w", value, err)
	}
	return EmailAddress{Value: addr.Address}, nil
}

// Email represents a composed email message ready for delivery.
type Email struct {
	From    EmailAddress
	To      []EmailAddress
	CC      []EmailAddress
	BCC     []EmailAddress
	Subject string
	// HTML is the HTML body of the email. At least one of HTML or Text must be set.
	HTML string
	// Text is the plain-text body. At least one of HTML or Text must be set.
	Text    string
	ReplyTo *EmailAddress
}

// Validate checks that the email satisfies minimum delivery requirements.
func (e Email) Validate() error {
	if len(e.To) == 0 {
		return fmt.Errorf("email must have at least one recipient")
	}
	if e.HTML == "" && e.Text == "" {
		return fmt.Errorf("email must have an HTML or text body")
	}
	if e.From.Value == "" {
		return fmt.Errorf("email must have a From address")
	}
	return nil
}

// EmailReceipt is returned by the provider after successful delivery.
type EmailReceipt struct {
	// MessageID is the provider-assigned message identifier.
	MessageID string
}

// TemplateEmailRequest requests delivery of a pre-defined template.
type TemplateEmailRequest struct {
	From         EmailAddress
	To           []EmailAddress
	TemplateName string
	Variables    map[string]any
}
```

- [ ] **Step 2: Create ports/files/port.go**

```go
// Package files defines the port interface for object/file storage.
package files

import (
	"context"
	"io"
	"net/url"
	"time"
)

// FileStorePort is the primary port for cloud object storage operations.
type FileStorePort interface {
	// Upload stores content under the given FileID.
	Upload(ctx context.Context, id FileID, content io.Reader, opts ...UploadOption) (UploadResult, error)
	// Download retrieves the content and metadata for a file.
	// The caller is responsible for closing FileObject.Content.
	Download(ctx context.Context, id FileID) (FileObject, error)
	// Delete removes a single file.
	Delete(ctx context.Context, id FileID) error
	// DeleteAll removes multiple files and reports which succeeded.
	DeleteAll(ctx context.Context, ids []FileID) (DeleteResult, error)
	// Exists returns true if the file exists, false if not found.
	Exists(ctx context.Context, id FileID) (bool, error)
	// GetMetadata returns file metadata without downloading content.
	GetMetadata(ctx context.Context, id FileID) (FileMetadata, error)
	// List returns files in a bucket under the given prefix.
	// prefix is path-like, e.g. "uploads/2026/" — no leading slash.
	List(ctx context.Context, bucket, prefix string, opts ...ListOption) (ListResult, error)
	// GeneratePresignedURL creates a time-limited URL for direct client access.
	GeneratePresignedURL(ctx context.Context, id FileID, op PresignedOperation, ttl time.Duration, opts ...PresignOption) (*url.URL, error)
	// Copy duplicates a file from src to dst within the same or different bucket.
	Copy(ctx context.Context, src, dst FileID) error
}

// FileID identifies a file by its bucket and key.
type FileID struct {
	Bucket string
	Key    string
}

// FileObject contains the content stream and metadata of a downloaded file.
// The caller must close Content after reading.
type FileObject struct {
	Content  io.ReadCloser
	Metadata FileMetadata
}

// FileMetadata holds descriptive information about a stored file.
type FileMetadata struct {
	ContentType  string
	ContentLength int64
	ETag         string
	LastModified time.Time
	UserMetadata map[string]string
}

// UploadResult is returned after a successful upload.
type UploadResult struct {
	ETag     string
	Location string
}

// DeleteResult reports the outcome of a bulk delete operation.
type DeleteResult struct {
	Deleted []FileID
	Failed  []DeleteError
}

// DeleteError pairs a FileID with the reason it could not be deleted.
type DeleteError struct {
	ID    FileID
	Cause error
}

// ListResult holds a page of listed files.
type ListResult struct {
	Objects         []FileMetadata
	ContinuationToken string
	IsTruncated     bool
}

// PresignedOperation is the HTTP method for a presigned URL.
type PresignedOperation string

const (
	PresignGet    PresignedOperation = "GET"
	PresignPut    PresignedOperation = "PUT"
	PresignDelete PresignedOperation = "DELETE"
)

// StorageClass controls durability/cost trade-offs in the storage backend.
type StorageClass string

const (
	StorageClassStandard StorageClass = "STANDARD"
	StorageClassGlacier  StorageClass = "GLACIER"
	StorageClassIA       StorageClass = "STANDARD_IA"
)

// UploadOption configures an upload operation.
type UploadOption func(*UploadOptions)

// UploadOptions holds resolved upload configuration.
type UploadOptions struct {
	ContentType  string
	StorageClass StorageClass
	Metadata     map[string]string
}

// WithContentType sets the MIME type for the uploaded file.
func WithContentType(ct string) UploadOption {
	return func(o *UploadOptions) { o.ContentType = ct }
}

// WithStorageClass sets the storage class for the uploaded file.
func WithStorageClass(sc StorageClass) UploadOption {
	return func(o *UploadOptions) { o.StorageClass = sc }
}

// WithMetadata attaches user-defined key-value metadata to the upload.
func WithMetadata(m map[string]string) UploadOption {
	return func(o *UploadOptions) { o.Metadata = m }
}

// ListOption configures a list operation.
type ListOption func(*ListOptions)

// ListOptions holds resolved list configuration.
type ListOptions struct {
	MaxKeys           int
	ContinuationToken string
}

// WithMaxKeys limits the number of objects returned in a list.
func WithMaxKeys(n int) ListOption {
	return func(o *ListOptions) { o.MaxKeys = n }
}

// PresignOption configures presigned URL generation.
type PresignOption func(*PresignOptions)

// PresignOptions holds resolved presign configuration.
type PresignOptions struct {
	ResponseContentDisposition string
}

// WithContentDisposition sets the Content-Disposition response header on the presigned URL.
func WithContentDisposition(cd string) PresignOption {
	return func(o *PresignOptions) { o.ResponseContentDisposition = cd }
}
```

- [ ] **Step 3: Build to verify compilation**

```bash
go build ./ports/...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add ports/email/ ports/files/
git commit -m "feat(ports): add email and files port interfaces"
```

---

## Task 6: ports/persistence and ports/template

**Files:**
- Create: `ports/persistence/repository.go`
- Create: `ports/template/port.go`

- [ ] **Step 1: Create ports/persistence/repository.go**

```go
// Package persistence defines repository port interfaces following DDD patterns.
// All methods accept context.Context as the first parameter.
package persistence

import "context"

// Repository is the base CRUD port for a domain entity E identified by ID.
// Save is an upsert — it may modify the entity (e.g. assign a generated ID or update timestamps).
type Repository[E any, ID any] interface {
	// Save persists the entity. Returns the saved entity (may differ from input).
	Save(ctx context.Context, entity E) (E, error)
	// FindByID returns (entity, true, nil) if found, (zero, false, nil) if not found,
	// or (zero, false, err) if a technical error occurred.
	FindByID(ctx context.Context, id ID) (E, bool, error)
	// DeleteByID removes the entity with the given ID. Not an error if not found.
	DeleteByID(ctx context.Context, id ID) error
	// Delete removes the entity. Not an error if not found.
	Delete(ctx context.Context, entity E) error
}

// PageableRepository extends Repository with paginated query support.
type PageableRepository[E any, ID any] interface {
	Repository[E, ID]
	// FindAll returns a page of entities matching the specification.
	FindAll(ctx context.Context, req PageRequest, spec Specification[E]) (PageResult[E], error)
	// Search returns a page of entities matching the specification, ordered by sort.
	Search(ctx context.Context, req PageRequest, spec Specification[E], sort Sort) (PageResult[E], error)
}

// Specification filters entities. Use Spec() for simple func-based specs.
// Implement the interface directly for specs that need SQL or Elasticsearch translation.
type Specification[E any] interface {
	// ToPredicate returns an in-memory filter function. Used by InMemoryRepository.
	ToPredicate() func(E) bool
}

// funcSpec wraps a plain function as a Specification.
type funcSpec[E any] struct{ fn func(E) bool }

func (s funcSpec[E]) ToPredicate() func(E) bool { return s.fn }

// Spec creates a Specification from a plain filter function.
// Use for in-memory and test scenarios. For production SQL/ES, implement Specification directly.
func Spec[E any](fn func(E) bool) Specification[E] {
	return funcSpec[E]{fn: fn}
}

// Sort defines the ordering for paginated queries.
type Sort struct {
	Field      string
	Descending bool
}

// PageRequest specifies which page to fetch. Pages are zero-indexed.
type PageRequest struct {
	Page int
	Size int
}

// PageResult is a paginated response containing a slice of entities.
type PageResult[E any] struct {
	Content       []E
	TotalElements int
	TotalPages    int
	Page          int
	Size          int
}
```

- [ ] **Step 2: Create ports/template/port.go**

```go
// Package template defines the port interface for server-side template rendering.
package template

import "context"

// TemplatePort renders named templates with provided data.
type TemplatePort interface {
	// Render executes the named template with the given data map.
	Render(ctx context.Context, name string, data map[string]any) (TemplateResult, error)
	// Exists reports whether a template with the given name is registered.
	Exists(ctx context.Context, name string) (bool, error)
}

// Content type constants for use in TemplateResult.
const (
	ContentTypeHTML = "text/html"
	ContentTypeText = "text/plain"
	ContentTypeXML  = "application/xml"
)

// TemplateResult holds the output of a rendered template.
type TemplateResult struct {
	TemplateName string
	Content      string
	// ContentType should be one of the ContentType* constants.
	ContentType string
	Charset     string
}

// HTMLResult constructs a TemplateResult with HTML content type.
func HTMLResult(name, content string) TemplateResult {
	return TemplateResult{TemplateName: name, Content: content, ContentType: ContentTypeHTML, Charset: "UTF-8"}
}

// TextResult constructs a TemplateResult with plain-text content type.
func TextResult(name, content string) TemplateResult {
	return TemplateResult{TemplateName: name, Content: content, ContentType: ContentTypeText, Charset: "UTF-8"}
}

// XMLResult constructs a TemplateResult with XML content type.
func XMLResult(name, content string) TemplateResult {
	return TemplateResult{TemplateName: name, Content: content, ContentType: ContentTypeXML, Charset: "UTF-8"}
}

// Bytes returns the Content as a UTF-8 byte slice.
func (t TemplateResult) Bytes() []byte { return []byte(t.Content) }

// IsEmpty returns true if the Content is empty.
func (t TemplateResult) IsEmpty() bool { return t.Content == "" }
```

- [ ] **Step 3: Build to verify**

```bash
go build ./ports/...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add ports/persistence/ ports/template/
git commit -m "feat(ports): add persistence repository and template port interfaces"
```

---

## Task 7: ports/observability — Field, Logger, Metrics, Tracer, conventions

**Files:**
- Create: `ports/observability/field.go`
- Create: `ports/observability/interfaces.go`
- Create: `ports/observability/conventions.go`

- [ ] **Step 1: Create ports/observability/field.go**

```go
// Package observability defines port interfaces for logging, metrics, and tracing.
// These interfaces are dependency-free — adapters (slog, prometheus, otel) implement them.
package observability

import "context"

// Field is a structured key-value pair used across logging and metrics APIs.
// Using the same type for both logging and metrics creates a consistent vocabulary.
type Field struct {
	Key   string
	Value any
}

// F creates a Field with the given key and value.
func F(key string, value any) Field { return Field{Key: key, Value: value} }

// Err creates a Field for error logging. Use instead of passing err as a separate parameter.
//
//	logger.Error(ctx, "failed to send email", obs.Err(err))
func Err(err error) Field { return F("error", err) }

// RequestID creates a Field for the current request identifier.
func RequestID(id string) Field { return F("request.id", id) }

// UserID creates a Field for the current user identifier.
func UserID(id string) Field { return F("user.id", id) }
```

- [ ] **Step 2: Create ports/observability/interfaces.go**

```go
package observability

import "context"

// Logger is the structured logging port. All methods accept a variadic list of Fields.
// Errors should always be passed as obs.Err(err) — never as a separate parameter.
type Logger interface {
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	// Error logs at error level. Pass the error via obs.Err(err) in the fields.
	Error(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)
}

// Counter tracks a monotonically increasing value.
type Counter interface {
	Inc()
	Add(v float64)
}

// Observer records observed values (e.g. durations, sizes).
type Observer interface {
	Observe(v float64)
}

// Metrics is the metrics port for counters and histograms.
// Labels use Field to avoid ordering bugs — label names and values are explicit.
type Metrics interface {
	Counter(name string, labels ...Field) Counter
	Histogram(name string, labels ...Field) Observer
}

// Span represents an active tracing span. Always call End() when the operation completes.
type Span interface {
	// End marks the span as complete. Must always be called (defer recommended).
	End()
	// RecordError attaches an error to the span. Aligned with OpenTelemetry API.
	RecordError(err error)
	// SetAttribute adds a key-value attribute to the span.
	SetAttribute(key string, value any)
}

// Tracer creates and manages tracing spans.
type Tracer interface {
	// StartSpan creates a new child span derived from ctx.
	// Always call span.End() when the operation is complete.
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}
```

- [ ] **Step 3: Create ports/observability/conventions.go**

```go
package observability

// Metric naming conventions — format: <domain>.<resource>.<operation>.<type>
// Use these constants to ensure consistent metric names across the codebase.
const (
	MetricRequestsTotal    = "app.requests.total"
	MetricRequestsDuration = "app.requests.duration_ms"

	MetricS3UploadsTotal   = "infra.s3.uploads.total"
	MetricS3DownloadsTotal = "infra.s3.downloads.total"

	MetricEmailSentTotal   = "infra.email.sent.total"
	MetricEmailFailedTotal = "infra.email.failed.total"

	MetricCacheHitsTotal   = "infra.cache.hits.total"
	MetricCacheMissesTotal = "infra.cache.misses.total"

	MetricOutboxProcessedTotal = "outbox.processed.total"
	MetricOutboxFailedTotal    = "outbox.failed.total"
	MetricOutboxLatencyMS      = "outbox.latency_ms"
)

// Attribute key conventions — format: <namespace>.<attribute>
// Use these constants for span attributes and structured log fields.
const (
	AttrRequestID    = "request.id"
	AttrUserID       = "user.id"
	AttrFileKey      = "file.key"
	AttrFileBucket   = "file.bucket"
	AttrEmailTo      = "email.to"
	AttrQueueTopic   = "queue.topic"
	AttrErrorCode    = "error.code"
	AttrErrorCategory = "error.category"
)
```

- [ ] **Step 4: Build to verify**

```bash
go build ./ports/...
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add ports/observability/
git commit -m "feat(ports): add observability port with Logger, Metrics, Tracer and Field helpers"
```

---

## Task 8: Remaining ports — cache, queue, sms, push, secrets, excel, compression

**Files:**
- Create: `ports/cache/port.go`
- Create: `ports/queue/port.go`
- Create: `ports/sms/port.go`
- Create: `ports/push/port.go`
- Create: `ports/secrets/port.go`
- Create: `ports/excel/port.go`
- Create: `ports/compression/port.go`

- [ ] **Step 1: Create ports/cache/port.go**

```go
// Package cache defines the port interface for distributed caching.
package cache

import (
	"context"
	"time"
)

// CachePort provides get/set/delete operations with TTL support.
type CachePort interface {
	// Get retrieves a cached value. Returns (value, true, nil) if found,
	// (nil, false, nil) if not found, or (nil, false, err) on error.
	Get(ctx context.Context, key string) (any, bool, error)
	// Set stores a value with the given TTL. TTL of 0 means no expiry.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Delete removes the cached value. Not an error if not found.
	Delete(ctx context.Context, key string) error
	// Exists returns true if the key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)
}
```

- [ ] **Step 2: Create ports/queue/port.go**

```go
// Package queue defines the port interface for message queue operations.
package queue

import "context"

// Message is a unit of work published to or received from a queue.
type Message struct {
	ID      string
	Topic   string
	Payload []byte
	// Attributes are optional key-value metadata attached to the message.
	Attributes map[string]string
}

// Handler processes a received message. Return an error to trigger redelivery.
type Handler func(ctx context.Context, msg Message) error

// QueuePort supports publishing and subscribing to topics.
type QueuePort interface {
	// Publish sends a message to the given topic.
	Publish(ctx context.Context, topic string, msg Message) error
	// Subscribe registers a handler for messages on the given topic.
	// The returned cancel function unregisters the handler.
	Subscribe(ctx context.Context, topic string, handler Handler) (cancel func(), err error)
	// Ping verifies connectivity to the queue broker.
	Ping(ctx context.Context) error
}
```

- [ ] **Step 3: Create ports/sms/port.go**

```go
// Package sms defines the port interface for SMS delivery.
package sms

import "context"

// SMSPort sends SMS messages via a configured provider.
type SMSPort interface {
	// Send delivers a text message to the given E.164 phone number.
	Send(ctx context.Context, to, body string) (SMSReceipt, error)
	// Ping verifies connectivity and credential validity.
	Ping(ctx context.Context) error
}

// SMSReceipt is returned by the provider after successful delivery.
type SMSReceipt struct {
	MessageID string
}
```

- [ ] **Step 4: Create ports/push/port.go**

```go
// Package push defines the port interface for push notification delivery.
package push

import "context"

// PushPort delivers push notifications to mobile devices or browsers.
type PushPort interface {
	// Send delivers a push notification.
	Send(ctx context.Context, notification PushNotification) (PushReceipt, error)
	// Ping verifies connectivity and credential validity.
	Ping(ctx context.Context) error
}

// PushNotification describes a push message to be delivered.
type PushNotification struct {
	// Token is the device/browser registration token.
	Token   string
	Title   string
	Body    string
	Data    map[string]string
	// Topic is an optional topic for fan-out delivery (provider-specific).
	Topic string
}

// PushReceipt is returned after successful delivery.
type PushReceipt struct {
	MessageID string
}
```

- [ ] **Step 5: Create ports/secrets/port.go**

```go
// Package secrets defines the port interface for secrets management.
package secrets

import (
	"context"
	"encoding/json"
)

// SecretsPort retrieves secrets from a secure store (e.g. AWS Secrets Manager, Vault).
type SecretsPort interface {
	// Get retrieves the secret value for the given key.
	Get(ctx context.Context, key string) (string, error)
	// GetJSON retrieves a JSON-encoded secret and unmarshals it into dest.
	GetJSON(ctx context.Context, key string, dest any) error
}

// ParseJSON is a helper for unmarshaling a secret string into a typed value.
func ParseJSON(secret string, dest any) error {
	return json.Unmarshal([]byte(secret), dest)
}
```

- [ ] **Step 6: Create ports/excel/port.go**

```go
// Package excel defines the port interface for Excel spreadsheet generation.
package excel

import (
	"context"
	"io"
)

// ExcelPort generates Excel (.xlsx) files from structured data.
type ExcelPort interface {
	// Generate produces an Excel file from the given request.
	// The returned io.Reader contains the .xlsx content.
	Generate(ctx context.Context, req ExcelRequest) (io.Reader, error)
}

// Sheet defines a single worksheet within an Excel workbook.
type Sheet struct {
	Name    string
	Headers []string
	Rows    [][]any
}

// ExcelRequest describes the workbook to generate.
type ExcelRequest struct {
	Filename string
	Sheets   []Sheet
}
```

- [ ] **Step 7: Create ports/compression/port.go**

```go
// Package compression defines the port interface for data compression.
package compression

import (
	"context"
	"io"
)

// Format identifies the compression algorithm.
type Format string

const (
	FormatGzip  Format = "gzip"
	FormatZstd  Format = "zstd"
	FormatSnappy Format = "snappy"
)

// CompressionPort compresses and decompresses data streams.
type CompressionPort interface {
	// Compress reads from src and returns a compressed stream in the given format.
	Compress(ctx context.Context, src io.Reader, format Format) (io.Reader, error)
	// Decompress reads a compressed stream and returns the decompressed data.
	Decompress(ctx context.Context, src io.Reader, format Format) (io.Reader, error)
}
```

- [ ] **Step 8: Build all ports**

```bash
go build ./ports/...
```

Expected: no errors

- [ ] **Step 9: Commit**

```bash
git add ports/
git commit -m "feat(ports): add cache, queue, sms, push, secrets, excel, compression port interfaces"
```

---

## Task 9: testkit/assert — AggregateAssertion

**Files:**
- Create: `testkit/assert/aggregate.go`

- [ ] **Step 1: Create testkit/assert/aggregate.go**

```go
// Package assert provides fluent assertion helpers for domain objects.
package assert

import (
	"testing"

	"github.com/marcusPrado02/go-commons/kernel/ddd"
)

// Eventful is the structural constraint for aggregates that expose domain events.
// Any struct with PullDomainEvents() satisfies this — no embedding of AggregateRoot required.
type Eventful interface {
	PullDomainEvents() []ddd.DomainEvent
}

// AggregateAssertion provides fluent assertions on domain aggregates.
type AggregateAssertion[T Eventful] struct {
	t      testing.TB
	actual T
	events []ddd.DomainEvent
}

// AssertAggregate begins a fluent assertion chain on the given aggregate.
// PullDomainEvents is called once and the result is held for all subsequent assertions.
func AssertAggregate[T Eventful](t testing.TB, actual T) *AggregateAssertion[T] {
	t.Helper()
	return &AggregateAssertion[T]{
		t:      t,
		actual: actual,
		events: actual.PullDomainEvents(),
	}
}

// HasDomainEvents asserts that exactly count events were raised.
func (a *AggregateAssertion[T]) HasDomainEvents(count int) *AggregateAssertion[T] {
	a.t.Helper()
	if len(a.events) != count {
		a.t.Errorf("expected %d domain events, got %d", count, len(a.events))
	}
	return a
}

// HasNoDomainEvents asserts that no events were raised.
func (a *AggregateAssertion[T]) HasNoDomainEvents() *AggregateAssertion[T] {
	return a.HasDomainEvents(0)
}

// HasEventOfType asserts that at least one event has the given EventType().
func (a *AggregateAssertion[T]) HasEventOfType(eventType string) *AggregateAssertion[T] {
	a.t.Helper()
	for _, e := range a.events {
		if e.EventType() == eventType {
			return a
		}
	}
	a.t.Errorf("no domain event of type %q found among %d events", eventType, len(a.events))
	return a
}

// FirstEventSatisfies asserts that the first event satisfies the predicate.
func (a *AggregateAssertion[T]) FirstEventSatisfies(fn func(ddd.DomainEvent) bool) *AggregateAssertion[T] {
	a.t.Helper()
	if len(a.events) == 0 {
		a.t.Error("no domain events to assert on")
		return a
	}
	if !fn(a.events[0]) {
		a.t.Errorf("first domain event of type %q did not satisfy predicate", a.events[0].EventType())
	}
	return a
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./testkit/...
```

Expected: no errors

- [ ] **Step 3: Write a smoke test for AggregateAssertion**

Create `testkit/assert/aggregate_test.go`:

```go
package assert_test

import (
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/kernel/ddd"
	"github.com/marcusPrado02/go-commons/testkit/assert"
)

type orderPlaced struct{ occurredAt time.Time }

func (e orderPlaced) EventType() string     { return "OrderPlaced" }
func (e orderPlaced) OccurredAt() time.Time { return e.occurredAt }

type order struct{ ddd.AggregateRoot[string] }

func TestAssertAggregate_HappyPath(t *testing.T) {
	o := order{AggregateRoot: ddd.NewAggregateRoot("order-1")}
	o.RegisterEvent(orderPlaced{occurredAt: time.Now()})

	assert.AssertAggregate(t, &o).
		HasDomainEvents(1).
		HasEventOfType("OrderPlaced").
		FirstEventSatisfies(func(e ddd.DomainEvent) bool {
			return e.EventType() == "OrderPlaced"
		})
}

func TestAssertAggregate_NoEvents(t *testing.T) {
	o := order{AggregateRoot: ddd.NewAggregateRoot("order-1")}
	assert.AssertAggregate(t, &o).HasNoDomainEvents()
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./testkit/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add testkit/assert/
git commit -m "feat(testkit): add AggregateAssertion fluent test helper"
```

---

## Task 10: testkit/contracts — RepositoryContract

**Files:**
- Create: `testkit/contracts/repository.go`

- [ ] **Step 1: Create testkit/contracts/repository.go**

```go
// Package contracts provides reusable test suites (contracts) for port implementations.
// Embed a contract suite in your adapter tests to verify it satisfies the port contract.
//
// Example:
//
//	func TestInMemoryRepository(t *testing.T) {
//	    suite.Run(t, &contracts.RepositoryContract[User, string]{
//	        Repo:         inmemory.NewInMemoryRepository[User, string](func(u User) string { return u.ID }),
//	        NewEntity:    func() User { return User{ID: uuid.New().String(), Name: "Alice"} },
//	        ExtractID:    func(u User) string { return u.ID },
//	        MutateEntity: func(u User) User { u.Name = "Bob"; return u },
//	    })
//	}
package contracts

import (
	"context"

	"github.com/marcusPrado02/go-commons/ports/persistence"
	"github.com/stretchr/testify/suite"
)

// RepositoryContract is a reusable test suite that verifies a Repository[E, ID] implementation
// satisfies the persistence port contract. Embed it in your adapter test file.
type RepositoryContract[E any, ID comparable] struct {
	suite.Suite
	// Repo is the repository under test. Set before running the suite.
	Repo persistence.Repository[E, ID]
	// NewEntity returns a new, unique entity for each call.
	NewEntity func() E
	// ExtractID extracts the identifier from an entity.
	ExtractID func(E) ID
	// MutateEntity returns a modified copy of the entity (e.g. change a name field).
	MutateEntity func(E) E
}

func (s *RepositoryContract[E, ID]) ctx() context.Context {
	return context.Background()
}

func (s *RepositoryContract[E, ID]) TestSave_InsertsNewEntity() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	id := s.ExtractID(saved)
	found, ok, err := s.Repo.FindByID(s.ctx(), id)
	s.Require().NoError(err)
	s.True(ok, "entity should be found after save")
	s.Equal(s.ExtractID(found), id)
}

func (s *RepositoryContract[E, ID]) TestSave_UpdatesExistingEntity() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	mutated := s.MutateEntity(saved)
	updated, err := s.Repo.Save(s.ctx(), mutated)
	s.Require().NoError(err)
	s.Equal(s.ExtractID(saved), s.ExtractID(updated), "ID should not change on update")
}

func (s *RepositoryContract[E, ID]) TestFindByID_Found() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	found, ok, err := s.Repo.FindByID(s.ctx(), s.ExtractID(saved))
	s.Require().NoError(err)
	s.True(ok)
	s.Equal(s.ExtractID(saved), s.ExtractID(found))
}

func (s *RepositoryContract[E, ID]) TestFindByID_NotFound() {
	entity := s.NewEntity()
	id := s.ExtractID(entity)

	_, ok, err := s.Repo.FindByID(s.ctx(), id)
	s.Require().NoError(err)
	s.False(ok, "unsaved entity should not be found")
}

func (s *RepositoryContract[E, ID]) TestDeleteByID_Removes() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	id := s.ExtractID(saved)
	err = s.Repo.DeleteByID(s.ctx(), id)
	s.Require().NoError(err)

	_, ok, err := s.Repo.FindByID(s.ctx(), id)
	s.Require().NoError(err)
	s.False(ok, "entity should not be found after delete")
}

func (s *RepositoryContract[E, ID]) TestDeleteByID_NotFoundIsNotError() {
	entity := s.NewEntity()
	id := s.ExtractID(entity)
	err := s.Repo.DeleteByID(s.ctx(), id)
	s.NoError(err, "deleting a non-existent entity should not return an error")
}

func (s *RepositoryContract[E, ID]) TestDelete_Removes() {
	entity := s.NewEntity()
	saved, err := s.Repo.Save(s.ctx(), entity)
	s.Require().NoError(err)

	err = s.Repo.Delete(s.ctx(), saved)
	s.Require().NoError(err)

	_, ok, err := s.Repo.FindByID(s.ctx(), s.ExtractID(saved))
	s.Require().NoError(err)
	s.False(ok)
}
```

- [ ] **Step 2: Build to verify**

```bash
go build ./testkit/...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add testkit/contracts/
git commit -m "feat(testkit): add RepositoryContract reusable test suite"
```

---

## Task 11: Full test run and coverage check

- [ ] **Step 1: Run all tests with race detector**

```bash
go test ./... -race -v
```

Expected: all PASS, no race conditions

- [ ] **Step 2: Check coverage**

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

Expected: total coverage >= 60%. Packages without tests (ports interface-only packages) show 0% — that is expected.

- [ ] **Step 3: Run linter**

```bash
golangci-lint run ./...
```

Expected: no issues. If `revive` flags exported types without comments, add godoc comments to the flagged types.

- [ ] **Step 4: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 5: Final commit**

```bash
git add go.mod go.sum go.work
git commit -m "chore: go mod tidy after core library implementation"
```

---

## Self-Review Checklist

After completing all tasks, verify:

- [ ] `go build ./...` passes with no errors
- [ ] `go test ./... -race` passes with no failures
- [ ] `golangci-lint run ./...` passes with no issues
- [ ] `kernel/errors`, `kernel/result`, `kernel/ddd` have >= 60% coverage
- [ ] `testkit/assert` has a smoke test
- [ ] All public types in `kernel/` and `ports/` have godoc comments
- [ ] No placeholder (`TODO`, `TBD`) in any `.go` file
- [ ] `go.work` includes only `.` (adapters are added in Plans 2 and 3)
