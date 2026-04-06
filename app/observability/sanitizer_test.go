package observability_test

import (
	"testing"

	"github.com/marcusPrado02/go-commons/app/observability"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSanitizer_RedactsSensitiveKeys(t *testing.T) {
	s := observability.NewDefaultSanitizer()

	sensitiveKeys := []string{"password", "token", "secret", "cpf", "credit_card", "authorization"}
	for _, key := range sensitiveKeys {
		result := s.Sanitize(key, "sensitive-value")
		assert.Equal(t, "[REDACTED]", result, "key %q should be redacted", key)
	}
}

func TestDefaultSanitizer_PassesThroughSafeKeys(t *testing.T) {
	s := observability.NewDefaultSanitizer()
	result := s.Sanitize("user_id", "u-123")
	assert.Equal(t, "u-123", result)
}

func TestDefaultSanitizer_CaseInsensitive(t *testing.T) {
	s := observability.NewDefaultSanitizer()
	assert.Equal(t, "[REDACTED]", s.Sanitize("PASSWORD", "val"))
	assert.Equal(t, "[REDACTED]", s.Sanitize("Token", "val"))
}

func TestDefaultSanitizer_SanitizeMap(t *testing.T) {
	s := observability.NewDefaultSanitizer()
	input := map[string]any{
		"username": "alice",
		"password": "s3cr3t",
		"token":    "abc123",
	}
	result := s.SanitizeMap(input)

	assert.Equal(t, "alice", result["username"])
	assert.Equal(t, "[REDACTED]", result["password"])
	assert.Equal(t, "[REDACTED]", result["token"])
}

func TestDefaultSanitizer_AdditionalKeys(t *testing.T) {
	s := observability.NewDefaultSanitizer("ssn", "api_key")
	assert.Equal(t, "[REDACTED]", s.Sanitize("ssn", "123-45-6789"))
	assert.Equal(t, "[REDACTED]", s.Sanitize("api_key", "sk-xyz"))
}
