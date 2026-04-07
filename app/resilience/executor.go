// Package resilience provides retry and circuit breaker capabilities.
// Use ResilienceExecutor.Run for void operations and Supply[T] for operations that return a value.
package resilience

import (
	"context"
	"fmt"
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

// ValidatePolicies returns an error if the ResiliencePolicySet has invalid configuration.
func ValidatePolicies(p ResiliencePolicySet) error {
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
