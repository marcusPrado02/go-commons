// Package secrets defines the port interface for secrets management.
package secrets

import (
	"context"
	"encoding/json"
)

// Port retrieves secrets from a secure store (e.g. AWS Secrets Manager, Vault).
type Port interface {
	// Get retrieves the secret value for the given key.
	Get(ctx context.Context, key string) (string, error)
	// GetJSON retrieves a JSON-encoded secret and unmarshals it into dest.
	GetJSON(ctx context.Context, key string, dest any) error
}

// ParseJSON is a helper for unmarshaling a secret string into a typed value.
func ParseJSON(secret string, dest any) error {
	return json.Unmarshal([]byte(secret), dest)
}
