// Example: Scheduler + Resilience
//
// This example demonstrates:
//   - Registering cron jobs with the Scheduler
//   - Wrapping a flaky external call with ResilienceExecutor (retry + circuit breaker)
//   - Using WithLogger for structured job lifecycle logging
//
// Run: go run ./examples/scheduler-resilience/
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marcusPrado02/go-commons/app/resilience"
	"github.com/marcusPrado02/go-commons/app/scheduler"
	obs "github.com/marcusPrado02/go-commons/ports/observability"
)

// --- Structured Logger adapter (slog-based) ----------------------------------

type slogLogger struct{ l *slog.Logger }

func (s *slogLogger) Debug(ctx context.Context, msg string, fields ...obs.Field) {
	s.l.DebugContext(ctx, msg, fieldsToArgs(fields)...)
}
func (s *slogLogger) Info(ctx context.Context, msg string, fields ...obs.Field) {
	s.l.InfoContext(ctx, msg, fieldsToArgs(fields)...)
}
func (s *slogLogger) Warn(ctx context.Context, msg string, fields ...obs.Field) {
	s.l.WarnContext(ctx, msg, fieldsToArgs(fields)...)
}
func (s *slogLogger) Error(ctx context.Context, msg string, fields ...obs.Field) {
	s.l.ErrorContext(ctx, msg, fieldsToArgs(fields)...)
}
func fieldsToArgs(fields []obs.Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		args = append(args, f.Key, f.Value)
	}
	return args
}

// --- Flaky external service simulator ----------------------------------------

var callCount int

func flakyExternalCall(ctx context.Context) error {
	callCount++
	// Fail 60% of the time to demonstrate retry behaviour.
	//nolint:gosec
	if rand.Float64() < 0.6 {
		return errors.New("service temporarily unavailable")
	}
	return nil
}

// --- Jobs -------------------------------------------------------------------

// SyncJob calls a flaky external service with retry + circuit breaker.
type SyncJob struct {
	exec     resilience.ResilienceExecutor
	policies resilience.ResiliencePolicySet
}

func (j *SyncJob) Name() string { return "external-sync" }

func (j *SyncJob) Run(ctx context.Context) error {
	return j.exec.Run(ctx, j.Name(), j.policies, flakyExternalCall)
}

// CleanupJob demonstrates a simple periodic cleanup task.
type CleanupJob struct{}

func (c *CleanupJob) Name() string { return "cleanup" }
func (c *CleanupJob) Run(_ context.Context) error {
	fmt.Printf("[cleanup] purged stale records at %s\n", time.Now().Format(time.TimeOnly))
	return nil
}

// --- Main -------------------------------------------------------------------

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := &slogLogger{slog.New(slog.NewTextHandler(os.Stdout, nil))}

	exec := resilience.NewExecutor(resilience.WithLogger(logger))
	policies := resilience.ResiliencePolicySet{
		RetryAttempts:   4,
		RetryDelay:      50 * time.Millisecond,
		RetryMaxDelay:   500 * time.Millisecond,
		TimeoutDuration: 3 * time.Second,
		CircuitBreaker: &resilience.CircuitBreakerConfig{
			MaxRequests:      3,
			Interval:         10 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 0.7,
		},
	}

	sched := scheduler.NewScheduler(
		scheduler.WithLogger(logger),
		scheduler.WithErrorHandler(func(job scheduler.Job, err error) {
			slog.Error("job failed", "job", job.Name(), "error", err)
		}),
	)

	// Run every 2 seconds.
	if err := sched.Register(&SyncJob{exec: exec, policies: policies}, "@every 2s"); err != nil {
		slog.Error("register sync job", "error", err)
		os.Exit(1)
	}
	// Run every 5 seconds.
	if err := sched.Register(&CleanupJob{}, "@every 5s"); err != nil {
		slog.Error("register cleanup job", "error", err)
		os.Exit(1)
	}

	sched.Start(ctx)
	fmt.Println("Scheduler started. Press Ctrl+C to stop.")

	<-ctx.Done()
	fmt.Println("Shutting down…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sched.Stop(shutdownCtx); err != nil {
		slog.Error("scheduler stop", "error", err)
	}
	fmt.Printf("Total external calls made: %d\n", callCount)
}
