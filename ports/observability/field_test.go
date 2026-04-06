package observability_test

import (
	"errors"
	"testing"

	"github.com/marcusPrado02/go-commons/ports/observability"
	"github.com/stretchr/testify/assert"
)

func TestF(t *testing.T) {
	f := observability.F("key", "value")
	assert.Equal(t, "key", f.Key)
	assert.Equal(t, "value", f.Value)
}

func TestErr(t *testing.T) {
	err := errors.New("boom")
	f := observability.Err(err)
	assert.Equal(t, "error", f.Key)
	assert.Equal(t, err, f.Value)
}

func TestRequestID(t *testing.T) {
	f := observability.RequestID("req-123")
	assert.Equal(t, "request.id", f.Key)
	assert.Equal(t, "req-123", f.Value)
}

func TestUserID(t *testing.T) {
	f := observability.UserID("user-456")
	assert.Equal(t, "user.id", f.Key)
	assert.Equal(t, "user-456", f.Value)
}
