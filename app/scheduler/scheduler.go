// Package scheduler provides a cron-based job scheduler with context propagation and panic recovery.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/robfig/cron/v3"
	obs "github.com/marcusPrado02/go-commons/ports/observability"
)

// Job is a named, runnable unit of work.
type Job interface {
	Name() string
	Run(ctx context.Context) error
}

// ErrorHandler is called when a job returns an error or panics.
type ErrorHandler func(job Job, err error)

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

// Option configures a Scheduler.
type Option func(*defaultScheduler)

// WithErrorHandler sets a custom handler for job errors and panics.
// If not set, errors are logged via slog and panics are logged with stack trace.
func WithErrorHandler(h ErrorHandler) Option {
	return func(s *defaultScheduler) { s.onError = h }
}

// WithLogger sets a structured logger for job lifecycle events (start, completion, errors).
func WithLogger(l obs.Logger) Option {
	return func(s *defaultScheduler) { s.logger = l }
}

type defaultScheduler struct {
	cron    *cron.Cron
	onError ErrorHandler
	logger  obs.Logger
}

// NewScheduler creates a Scheduler using standard cron expressions plus descriptors (@every, @hourly, etc.).
func NewScheduler(opts ...Option) Scheduler {
	s := &defaultScheduler{
		cron: cron.New(cron.WithSeconds()),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Register validates the schedule expression and adds the job.
func (s *defaultScheduler) Register(job Job, schedule string) error {
	p := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := p.Parse(schedule); err != nil {
		return fmt.Errorf("scheduler: invalid cron expression %q for job %q: %w", schedule, job.Name(), err)
	}

	_, err := s.cron.AddFunc(schedule, func() {
		s.runSafe(job)
	})
	return err
}

func (s *defaultScheduler) runSafe(job Job) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic: %v\n%s", r, debug.Stack())
			s.handleError(job, err)
		}
	}()
	if s.logger != nil {
		s.logger.Info(ctx, "scheduler: job started", obs.F("job", job.Name()))
	}
	start := time.Now()
	if err := job.Run(ctx); err != nil {
		s.handleError(job, err)
		return
	}
	if s.logger != nil {
		s.logger.Info(ctx, "scheduler: job completed",
			obs.F("job", job.Name()),
			obs.F("duration_ms", time.Since(start).Milliseconds()),
		)
	}
}

func (s *defaultScheduler) handleError(job Job, err error) {
	if s.onError != nil {
		s.onError(job, err)
		return
	}
	slog.Error("scheduler: job error", "job", job.Name(), "error", err)
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
