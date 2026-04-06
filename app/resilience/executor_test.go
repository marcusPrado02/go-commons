package resilience_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/app/resilience"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_Run_Success(t *testing.T) {
	exec := resilience.NewExecutor()
	called := 0

	err := exec.Run(context.Background(), "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond,
	}, func(ctx context.Context) error {
		called++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, called, "should succeed on first attempt")
}

func TestExecutor_Run_RetriesOnError(t *testing.T) {
	exec := resilience.NewExecutor()
	var attempts int32

	err := exec.Run(context.Background(), "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 3,
		RetryDelay:    time.Millisecond,
		RetryMaxDelay: 10 * time.Millisecond,
	}, func(ctx context.Context) error {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), attempts, "should retry until success")
}

func TestExecutor_Run_ExhaustsRetries(t *testing.T) {
	exec := resilience.NewExecutor()
	var attempts int32

	err := exec.Run(context.Background(), "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 2,
		RetryDelay:    time.Millisecond,
	}, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("permanent error")
	})

	assert.Error(t, err)
	assert.Equal(t, int32(3), attempts, "1 initial + 2 retries")
}

func TestExecutor_Run_RespectsContextCancellation(t *testing.T) {
	exec := resilience.NewExecutor()
	ctx, cancel := context.WithCancel(context.Background())
	var attempts int32

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := exec.Run(ctx, "test-op", resilience.ResiliencePolicySet{
		RetryAttempts: 10,
		RetryDelay:    20 * time.Millisecond,
	}, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Less(t, atomic.LoadInt32(&attempts), int32(10), "context cancel should stop retries early")
}

func TestSupply_ReturnsValue(t *testing.T) {
	exec := resilience.NewExecutor()

	result, err := resilience.Supply(context.Background(), exec, "get-user", resilience.ResiliencePolicySet{
		RetryAttempts: 1,
	}, func(ctx context.Context) (string, error) {
		return "user-123", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "user-123", result)
}

func TestSupply_ReturnsZeroOnError(t *testing.T) {
	exec := resilience.NewExecutor()

	result, err := resilience.Supply(context.Background(), exec, "get-user", resilience.ResiliencePolicySet{
		RetryAttempts: 0,
	}, func(ctx context.Context) (string, error) {
		return "", errors.New("not found")
	})

	assert.Error(t, err)
	assert.Empty(t, result)
}
