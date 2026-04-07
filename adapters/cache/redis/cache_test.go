package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	rediscache "github.com/marcusPrado02/go-commons/adapters/cache/redis"
)

func newTestCache(t *testing.T) (*rediscache.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return rediscache.NewFromClient(client), mr
}

func TestCache_SetGet_RoundTrip(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "greeting", "hello", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, ok, err := c.Get(ctx, "greeting")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != "hello" {
		t.Errorf("expected %q, got %v", "hello", val)
	}
}

func TestCache_Get_Miss(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	val, ok, err := c.Get(ctx, "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok || val != nil {
		t.Errorf("expected miss, got ok=%v val=%v", ok, val)
	}
}

func TestCache_Delete(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "key", 42.0, 0)
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, ok, _ := c.Get(ctx, "key")
	if ok {
		t.Error("key should have been deleted")
	}

	// Delete of missing key must not return an error.
	if err := c.Delete(ctx, "nonexistent"); err != nil {
		t.Errorf("Delete missing key: %v", err)
	}
}

func TestCache_Exists(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	exists, err := c.Exists(ctx, "k")
	if err != nil || exists {
		t.Fatalf("expected (false, nil), got (%v, %v)", exists, err)
	}

	_ = c.Set(ctx, "k", true, 0)
	exists, err = c.Exists(ctx, "k")
	if err != nil || !exists {
		t.Fatalf("expected (true, nil), got (%v, %v)", exists, err)
	}
}

func TestCache_TTL_Expiry(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "temp", "bye", 50*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Fast-forward miniredis clock so the key expires.
	mr.FastForward(100 * time.Millisecond)

	_, ok, err := c.Get(ctx, "temp")
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if ok {
		t.Error("key should have expired")
	}
}

func TestCache_Set_JSONTypes(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	_ = c.Set(ctx, "num", 3.14, 0)
	val, ok, _ := c.Get(ctx, "num")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val.(float64) != 3.14 {
		t.Errorf("expected 3.14, got %v", val)
	}
}
