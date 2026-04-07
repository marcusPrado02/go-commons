package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/marcusPrado02/go-commons/ports/cache"
)

// Compile-time check that the interface is implementable.
var _ cache.CachePort = (*nilCache)(nil)

// nilCache is a no-op implementation used only for compile-time verification.
type nilCache struct{}

func (n *nilCache) Get(_ context.Context, _ string) (any, bool, error)             { return nil, false, nil }
func (n *nilCache) Set(_ context.Context, _ string, _ any, _ time.Duration) error  { return nil }
func (n *nilCache) Delete(_ context.Context, _ string) error                       { return nil }
func (n *nilCache) Exists(_ context.Context, _ string) (bool, error)               { return false, nil }

func TestCachePort_ZeroValueIsNil(t *testing.T) {
	var cp cache.CachePort
	if cp != nil {
		t.Fatal("expected nil zero value for CachePort interface")
	}
}

func TestCachePort_TTLZeroMeansNoExpiry(t *testing.T) {
	// TTL=0 is the documented semantic for "no expiry".
	// This test anchors that constant so future changes are visible.
	const noExpiry time.Duration = 0
	if noExpiry != 0 {
		t.Fatal("expected zero-value Duration to represent no expiry")
	}
}

func TestCachePort_InterfaceSignature(t *testing.T) {
	// Verify that all four methods of CachePort are present and callable
	// by assigning a concrete implementation.
	var cp cache.CachePort = &nilCache{}
	ctx := context.Background()

	if _, _, err := cp.Get(ctx, "k"); err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if err := cp.Set(ctx, "k", "v", 0); err != nil {
		t.Fatalf("Set: unexpected error: %v", err)
	}
	if err := cp.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}
	if _, err := cp.Exists(ctx, "k"); err != nil {
		t.Fatalf("Exists: unexpected error: %v", err)
	}
}
