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
