package observability

import "strings"

const redacted = "[REDACTED]"

// defaultSensitiveKeys are always redacted regardless of configuration.
var defaultSensitiveKeys = []string{
	"password", "token", "secret", "cpf", "credit_card", "authorization",
	"api_key", "private_key", "access_token", "refresh_token", "ssn", "cnpj", "rg",
}

// LogSanitizer removes PII and secrets from structured log fields before emission.
type LogSanitizer interface {
	Sanitize(key string, value any) any
	SanitizeMap(input map[string]any) map[string]any
}

type defaultSanitizer struct {
	sensitiveKeys map[string]struct{}
}

// NewDefaultSanitizer creates a LogSanitizer that redacts common sensitive field names.
// additionalKeys adds extra keys to the default redaction list (case-insensitive).
func NewDefaultSanitizer(additionalKeys ...string) LogSanitizer {
	keys := make(map[string]struct{}, len(defaultSensitiveKeys)+len(additionalKeys))
	for _, k := range defaultSensitiveKeys {
		keys[strings.ToLower(k)] = struct{}{}
	}
	for _, k := range additionalKeys {
		keys[strings.ToLower(k)] = struct{}{}
	}
	return &defaultSanitizer{sensitiveKeys: keys}
}

// Sanitize returns "[REDACTED]" if the key is sensitive, otherwise returns value unchanged.
// If value is a map[string]any, it is recursively sanitized.
func (s *defaultSanitizer) Sanitize(key string, value any) any {
	if _, sensitive := s.sensitiveKeys[strings.ToLower(key)]; sensitive {
		return redacted
	}
	if nested, ok := value.(map[string]any); ok {
		return s.SanitizeMap(nested)
	}
	return value
}

// SanitizeMap returns a new map with sensitive values replaced by "[REDACTED]".
// Nested map[string]any values are recursively sanitized.
func (s *defaultSanitizer) SanitizeMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for k, v := range input {
		result[k] = s.Sanitize(k, v)
	}
	return result
}
