// Package cache defines the port interface for distributed caching.
package cache

import (
	"context"
	"time"
)

// CachePort provides get/set/delete operations with TTL support.
type CachePort interface {
	// Get retrieves a cached value. Returns (value, true, nil) if found,
	// (nil, false, nil) if not found, or (nil, false, err) on error.
	Get(ctx context.Context, key string) (any, bool, error)
	// Set stores a value with the given TTL. TTL of 0 means no expiry.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	// Delete removes the cached value. Not an error if not found.
	Delete(ctx context.Context, key string) error
	// Exists returns true if the key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)
}
