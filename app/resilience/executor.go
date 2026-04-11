// Package resilience provides retry and circuit breaker capabilities.
// Use Executor.Run for void operations and Supply[T] for operations that return a value.
package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/sony/gobreaker"
	obs "github.com/marcusPrado02/go-commons/ports/observability"
)

// PolicySet configures the retry and circuit breaker behavior.
type PolicySet struct {
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

// Executor executes actions with retry and circuit breaker policies applied.
type Executor interface {
	Run(ctx context.Context, name string, policies PolicySet, action func(ctx context.Context) error) error
}

// ExecutorOption configures a Executor.
type ExecutorOption func(*defaultExecutor)

// WithLogger sets a structured logger for retry attempts and circuit breaker events.
func WithLogger(l obs.Logger) ExecutorOption {
	return func(e *defaultExecutor) { e.logger = l }
}

type defaultExecutor struct {
	logger obs.Logger
}

// NewExecutor creates a new Executor.
func NewExecutor(opts ...ExecutorOption) Executor {
	e := &defaultExecutor{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ValidatePolicies returns an error if the PolicySet has invalid configuration.
func ValidatePolicies(p PolicySet) error {
	if p.RetryAttempts < 0 {
		return fmt.Errorf("resilience: RetryAttempts must be >= 0, got %d", p.RetryAttempts)
	}
	if p.RetryDelay < 0 {
		return fmt.Errorf("resilience: RetryDelay must be >= 0, got %v", p.RetryDelay)
	}
	if p.RetryMaxDelay > 0 && p.RetryMaxDelay < p.RetryDelay {
		return fmt.Errorf("resilience: RetryMaxDelay (%v) must be >= RetryDelay (%v)", p.RetryMaxDelay, p.RetryDelay)
	}
	if p.CircuitBreaker != nil {
		if p.CircuitBreaker.FailureThreshold < 0 || p.CircuitBreaker.FailureThreshold > 1 {
			return fmt.Errorf("resilience: CircuitBreaker.FailureThreshold must be in [0, 1], got %v", p.CircuitBreaker.FailureThreshold)
		}
	}
	return nil
}

func (e *defaultExecutor) Run(ctx context.Context, name string, policies PolicySet, action func(ctx context.Context) error) error {
	if err := ValidatePolicies(policies); err != nil {
		return err
	}
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
		cancelAttempt := func() {}
		if policies.TimeoutDuration > 0 {
			attemptCtx, cancelAttempt = context.WithTimeout(ctx, policies.TimeoutDuration)
		}
		lastErr = run(attemptCtx)
		cancelAttempt()
		if lastErr == nil {
			return nil
		}

		if attempt < policies.RetryAttempts {
			delay := jitterDelay(attempt, policies.RetryDelay, policies.RetryMaxDelay)
			if e.logger != nil {
				e.logger.Warn(ctx, "resilience: retrying after error",
					obs.F("name", name),
					obs.F("attempt", attempt+1),
					obs.F("max_attempts", policies.RetryAttempts),
					obs.F("delay_ms", delay.Milliseconds()),
					obs.Err(lastErr),
				)
			}
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
	window := exp
	if maxDelay > 0 && exp > maxDelay {
		window = maxDelay
	}
	//nolint:gosec // math/rand is fine for jitter — not a security-sensitive operation
	return time.Duration(rand.Int63n(int64(window) + 1))
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
func Supply[T any](ctx context.Context, exec Executor, name string, policies PolicySet, action func(ctx context.Context) (T, error)) (T, error) {
	var result T
	err := exec.Run(ctx, name, policies, func(ctx context.Context) error {
		var err error
		result, err = action(ctx)
		return err
	})
	return result, err
}
