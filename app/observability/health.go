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

// timedCheck wraps a HealthCheck with a per-check execution timeout.
type timedCheck struct {
	inner   HealthCheck
	timeout time.Duration
}

// WithCheckTimeout wraps a HealthCheck to enforce a per-check execution deadline.
// If the check exceeds d, it returns StatusDown with a "timed_out" detail.
func WithCheckTimeout(check HealthCheck, d time.Duration) HealthCheck {
	return &timedCheck{inner: check, timeout: d}
}

func (t *timedCheck) Name() string          { return t.inner.Name() }
func (t *timedCheck) Type() HealthCheckType { return t.inner.Type() }
func (t *timedCheck) Check(ctx context.Context) HealthCheckResult {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	done := make(chan HealthCheckResult, 1)
	go func() { done <- t.inner.Check(ctx) }()
	select {
	case result := <-done:
		return result
	case <-ctx.Done():
		return HealthCheckResult{
			Status:  StatusDown,
			Details: map[string]any{"error": "timed out after " + t.timeout.String()},
		}
	}
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
