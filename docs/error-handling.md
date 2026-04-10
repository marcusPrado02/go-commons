# Error Handling

go-commons provides a layered error system designed around three goals:

1. **Rich domain context** — errors carry code, category, severity, and structured details.
2. **Immutability** — `Problem` is a value type; `With*` builders always return a new copy.
3. **Standard compatibility** — `Problem` implements `error` and supports `errors.Is`/`errors.As` via `Unwrap`.

---

## Problem

`Problem` is the primary error type. It replaces plain `fmt.Errorf` in domain and application code.

```go
import kerrors "github.com/marcusPrado02/go-commons/kernel/errors"

// Use a sentinel error as a base and add context:
return kerrors.ErrNotFound.
    WithDetail("resource", "Order").
    WithDetail("id", orderID)

// Or create a custom error:
var ErrInsufficientStock = kerrors.NewProblem(
    "INSUFFICIENT_STOCK",
    kerrors.CategoryBusiness,
    kerrors.SeverityWarning,
    "not enough stock to fulfil order",
)

return ErrInsufficientStock.
    WithDetail("requested", qty).
    WithDetail("available", stock).
    WithCause(originalErr)
```

### Sentinels

| Variable | Code | Category | When to use |
|---|---|---|---|
| `ErrNotFound` | `NOT_FOUND` | `NOT_FOUND` | Resource does not exist |
| `ErrUnauthorized` | `UNAUTHORIZED` | `UNAUTHORIZED` | Auth/permission failure |
| `ErrValidation` | `VALIDATION` | `VALIDATION` | Input validation failure |
| `ErrTechnical` | `TECHNICAL` | `TECHNICAL` | Infrastructure/unexpected error |

### Builder Methods

```go
p := kerrors.ErrTechnical

// Add a single key-value detail:
p = p.WithDetail("operation", "db.query")

// Merge a map of details in one call:
p = p.WithDetails(map[string]any{
    "table":   "orders",
    "queryMs": 1200,
})

// Attach the underlying cause (preserved by errors.Unwrap):
p = p.WithCause(pgErr)
```

### Checking Errors

```go
var prob kerrors.Problem
if errors.As(err, &prob) {
    switch prob.Category {
    case kerrors.CategoryNotFound:
        // return 404
    case kerrors.CategoryValidation:
        // return 400
    default:
        // return 500
    }
}
```

---

## ErrorCode

`ErrorCode` is a strongly-typed string for machine-readable error identification.

```go
code, err := kerrors.NewErrorCode("ORDER_CANCELLED")
if err != nil {
    // code was empty
}
```

Use `ErrorCode` when building event-driven systems where consumers need to discriminate errors without parsing strings.

---

## Result[T]

`Result[T]` is an opt-in functional wrapper for operations that return a value. Use it when chaining multiple transformations is more expressive than sequential `if err != nil` checks.

```go
import "github.com/marcusPrado02/go-commons/kernel/result"

// Wrapping a (T, error) pair:
r := result.FromError(repo.FindByID(ctx, id))

// Transforming the value if successful:
nameResult := result.Map(r, func(u User) string { return u.Name })

// Consuming:
name := nameResult.Or("anonymous")       // fallback on failure
name  = nameResult.OrElse(defaultName)   // lazy fallback
name  = nameResult.Value()               // panics if failed — only when IsOk() is guaranteed

// Chaining two fallible operations:
enriched := result.FlatMap(r, func(u User) result.Result[EnrichedUser] {
    return result.FromError(enrich(ctx, u))
})

// Back to idiomatic Go:
user, err := enriched.Unwrap()
```

### When to Use Result[T] vs (T, error)

- **Port interfaces always use `(T, error)`** — standard Go, easier to chain with `if err != nil`.
- **Use `Result[T]` in application/domain logic** where you have multiple chained transformations and want to avoid deeply nested error checks.
- Never return `Result[T]` from a public API of an adapter.

---

## Propagating Errors Between Layers

**Adapters** wrap SDK errors into `Problem` before returning:

```go
if awsErr != nil {
    return kerrors.ErrTechnical.
        WithDetail("operation", "s3.PutObject").
        WithCause(awsErr)
}
```

**Application layer** passes errors through without wrapping unless adding context:

```go
if err := port.Send(ctx, email); err != nil {
    // Only wrap if adding meaningful context:
    return fmt.Errorf("outbox: deliver message %s: %w", msg.ID, err)
}
```

**Domain layer** uses `Problem` directly, never infrastructure types.

---

## Security: Never Log Details in Production Without Sanitization

`Problem.Details` may contain sensitive data (user IDs, query parameters, tokens). Always run details through `app/observability.LogSanitizer` before logging:

```go
sanitized := sanitizer.Sanitize(prob.Details)
logger.Error(ctx, prob.Message, obs.F("details", sanitized))
```

See [docs/security.md](security.md) for the full list of automatically redacted keys.
