// Package redis provides a CachePort implementation backed by Redis via go-redis/v9.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/marcusPrado02/go-commons/ports/cache"
)

// Cache implements cache.CachePort using a Redis backend.
// Values are serialised to JSON strings, so any JSON-marshallable type is supported.
type Cache struct {
	client *goredis.Client
}

// Option configures a Cache.
type Option func(*goredis.Options)

// WithPassword sets the Redis password.
func WithPassword(password string) Option {
	return func(o *goredis.Options) { o.Password = password }
}

// WithDB selects the Redis logical database (0-15).
func WithDB(db int) Option {
	return func(o *goredis.Options) { o.DB = db }
}

// New creates a Cache connected to the given Redis address (e.g. "localhost:6379").
func New(addr string, opts ...Option) *Cache {
	o := &goredis.Options{Addr: addr}
	for _, opt := range opts {
		opt(o)
	}
	return &Cache{client: goredis.NewClient(o)}
}

// NewFromClient creates a Cache from an existing *goredis.Client.
// Useful for sharing a connection pool or injecting a miniredis client in tests.
func NewFromClient(client *goredis.Client) *Cache {
	return &Cache{client: client}
}

// Get retrieves the cached value for key.
// Returns (value, true, nil) on hit, (nil, false, nil) on miss, (nil, false, err) on error.
func (c *Cache) Get(ctx context.Context, key string) (any, bool, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis: get %q: %w", key, err)
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false, fmt.Errorf("redis: unmarshal %q: %w", key, err)
	}
	return value, true, nil
}

// Set stores value under key with the given TTL. TTL of 0 means no expiry.
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redis: marshal %q: %w", key, err)
	}
	if err := c.client.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("redis: set %q: %w", key, err)
	}
	return nil
}

// Delete removes the cached value for key. Not an error if the key does not exist.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: delete %q: %w", key, err)
	}
	return nil
}

// Exists reports whether key is present in the cache.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis: exists %q: %w", key, err)
	}
	return n > 0, nil
}

// Close releases the Redis connection pool.
func (c *Cache) Close() error { return c.client.Close() }

var _ cache.CachePort = (*Cache)(nil)
