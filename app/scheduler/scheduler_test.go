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
	name  string
	count atomic.Int32
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

	err := s.Register(job, "@every 1s")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	time.Sleep(2100 * time.Millisecond)
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
