// Package ratelimit defines the port interface for rate limiting.
package ratelimit

import "context"

// RateLimiter controls the rate of operations per key (e.g. user ID, IP address).
type RateLimiter interface {
	// Allow returns true immediately if the key is within its rate limit.
	// Returns false without blocking if the limit has been exceeded.
	Allow(ctx context.Context, key string) (bool, error)
	// Wait blocks until the key is allowed to proceed or ctx is cancelled.
	// Returns ctx.Err() if the context is cancelled before permission is granted.
	Wait(ctx context.Context, key string) error
}
