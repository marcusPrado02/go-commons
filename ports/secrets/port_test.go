package secrets_test

import (
	"testing"

	"github.com/marcusPrado02/go-commons/ports/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJSON(t *testing.T) {
	type config struct{ Host string }
	var cfg config
	err := secrets.ParseJSON(`{"Host":"localhost"}`, &cfg)
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
}

func TestParseJSON_invalid(t *testing.T) {
	var cfg struct{ X int }
	err := secrets.ParseJSON(`not json`, &cfg)
	assert.Error(t, err)
}
