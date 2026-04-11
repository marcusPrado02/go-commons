package contracts

import (
	"context"
	"time"

	"github.com/marcusPrado02/go-commons/ports/cache"
	"github.com/stretchr/testify/suite"
)

// CacheContract is a reusable test suite for Port implementations.
//
// Example:
//
//	func TestRedisCache(t *testing.T) {
//	    suite.Run(t, &contracts.CacheContract{Cache: myredis.New(...)})
//	}
type CacheContract struct {
	suite.Suite
	// Cache is the Port implementation under test.
	Cache cache.Port
}

func (s *CacheContract) TestSet_Get_RoundTrip() {
	ctx := context.Background()
	s.Require().NoError(s.Cache.Set(ctx, "k1", "hello", 0))

	val, ok, err := s.Cache.Get(ctx, "k1")
	s.Require().NoError(err)
	s.True(ok, "expected key to exist after Set")
	s.Equal("hello", val)
}

func (s *CacheContract) TestGet_MissingKey_ReturnsNotFound() {
	ctx := context.Background()
	_, ok, err := s.Cache.Get(ctx, "no-such-key-xyz")
	s.Require().NoError(err)
	s.False(ok, "expected missing key to return false")
}

func (s *CacheContract) TestDelete_RemovesKey() {
	ctx := context.Background()
	s.Require().NoError(s.Cache.Set(ctx, "del-key", 42, 0))
	s.Require().NoError(s.Cache.Delete(ctx, "del-key"))

	_, ok, err := s.Cache.Get(ctx, "del-key")
	s.Require().NoError(err)
	s.False(ok, "key should be gone after Delete")
}

func (s *CacheContract) TestDelete_MissingKey_NoError() {
	s.Require().NoError(s.Cache.Delete(context.Background(), "never-set-xyz"))
}

func (s *CacheContract) TestExists_TrueAfterSet() {
	ctx := context.Background()
	s.Require().NoError(s.Cache.Set(ctx, "exists-key", true, 0))

	ok, err := s.Cache.Exists(ctx, "exists-key")
	s.Require().NoError(err)
	s.True(ok)
}

func (s *CacheContract) TestExists_FalseForMissingKey() {
	ok, err := s.Cache.Exists(context.Background(), "missing-xyz")
	s.Require().NoError(err)
	s.False(ok)
}

func (s *CacheContract) TestSet_TTL_ExpiresEntry() {
	ctx := context.Background()
	s.Require().NoError(s.Cache.Set(ctx, "ttl-key", "temporary", 50*time.Millisecond))

	time.Sleep(100 * time.Millisecond)

	_, ok, err := s.Cache.Get(ctx, "ttl-key")
	s.Require().NoError(err)
	s.False(ok, "expected key to expire after TTL")
}
