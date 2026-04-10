# Resilience

`app/resilience` provides retry with exponential backoff and a circuit breaker, composable via `ResiliencePolicySet`.

---

## ResiliencePolicySet

```go
import "github.com/marcusPrado02/go-commons/app/resilience"

policies := resilience.ResiliencePolicySet{
    RetryAttempts:   3,                // retries after the first attempt (0 = no retry)
    RetryDelay:      100 * time.Millisecond,
    RetryMaxDelay:   2 * time.Second,
    TimeoutDuration: 5 * time.Second,  // per-attempt timeout (0 = none)
    CircuitBreaker: &resilience.CircuitBreakerConfig{
        MaxRequests:      5,
        Interval:         10 * time.Second,
        Timeout:          30 * time.Second,
        FailureThreshold: 0.5, // trip at 50% failure rate
    },
}
```

Policies are validated at the start of each `Run` call. Invalid configurations (e.g. `RetryAttempts < 0`, `RetryMaxDelay < RetryDelay`) return an error immediately.

---

## Retry with Jitter Backoff

The retry strategy uses **full jitter**:

```
delay = random(0, min(RetryMaxDelay, RetryDelay × 2^attempt))
```

Full jitter spreads retries uniformly across the window, preventing **thundering herd** — the situation where all clients retry at the same time after a service recovers.

| Attempt | Base (100ms) | Max (2s) | Jitter range |
|---|---|---|---|
| 1 | 200ms | 200ms | 0–200ms |
| 2 | 400ms | 400ms | 0–400ms |
| 3 | 800ms | 800ms | 0–800ms |
| 4 | 1600ms | 2000ms | 0–2000ms |

---

## Circuit Breaker States

The circuit breaker is powered by `gobreaker` and follows a three-state machine:

```
         failures ≥ threshold
  CLOSED ──────────────────────► OPEN
    ▲                               │
    │    success                    │ timeout elapsed
    │◄─────────────── HALF-OPEN ◄──┘
```

| State | Behaviour |
|---|---|
| **CLOSED** | All requests pass through; failure counts are tracked |
| **OPEN** | All requests are rejected immediately with `gobreaker.ErrOpenState` |
| **HALF-OPEN** | Up to `MaxRequests` are allowed through as probes |

The circuit trips when `TotalFailures / Requests ≥ FailureThreshold` AND `Requests ≥ 5` (hardcoded minimum to avoid tripping on a single failure at startup).

---

## Using the Executor

```go
exec := resilience.NewExecutor(
    resilience.WithLogger(logger),
)

// Void operation:
err := exec.Run(ctx, "payment.charge", policies, func(ctx context.Context) error {
    return paymentClient.Charge(ctx, amount)
})

// Operation with return value:
result, err := resilience.Supply(ctx, exec, "user.fetch", policies,
    func(ctx context.Context) (User, error) {
        return userRepo.FindByID(ctx, id)
    },
)
```

---

## Validating Policies Ahead of Time

`ValidatePolicies` is exported for use in constructors or startup:

```go
if err := resilience.ValidatePolicies(policies); err != nil {
    return fmt.Errorf("invalid resilience config: %w", err)
}
```

`Run` also calls `ValidatePolicies` internally, so invalid configs are caught even if you skip this step.

---

## ResiliencePolicySet Reference

| Field | Type | Default | Description |
|---|---|---|---|
| `RetryAttempts` | `int` | `0` | Retries after the first attempt. `0` = single attempt, no retry |
| `RetryDelay` | `time.Duration` | `0` | Base delay before first retry |
| `RetryMaxDelay` | `time.Duration` | `0` | Cap on backoff. `0` = no cap |
| `TimeoutDuration` | `time.Duration` | `0` | Per-attempt context deadline. `0` = no timeout |
| `CircuitBreaker` | `*CircuitBreakerConfig` | `nil` | Omit to disable circuit breaking |

### CircuitBreakerConfig Reference

| Field | Type | Description |
|---|---|---|
| `MaxRequests` | `uint32` | Requests allowed in HALF-OPEN state |
| `Interval` | `time.Duration` | Cyclic period for resetting failure counts in CLOSED state |
| `Timeout` | `time.Duration` | How long the circuit stays OPEN before transitioning to HALF-OPEN |
| `FailureThreshold` | `float64` | Failure ratio [0.0–1.0] that trips the circuit |

---

## Example: Resilient Database Query

```go
var user User
err := exec.Run(ctx, "user.findByEmail", resilience.ResiliencePolicySet{
    RetryAttempts:   2,
    RetryDelay:      50 * time.Millisecond,
    RetryMaxDelay:   500 * time.Millisecond,
    TimeoutDuration: 3 * time.Second,
}, func(ctx context.Context) error {
    var err error
    user, _, err = userRepo.FindByID(ctx, id)
    return err
})
```
