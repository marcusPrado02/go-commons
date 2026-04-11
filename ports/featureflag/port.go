// Package featureflag defines the port interface for feature toggles.
package featureflag

import "context"

// Port controls feature availability per user or context.
// Implementations may be backed by environment variables, a remote service
// (LaunchDarkly, Unleash), or a database table.
type Port interface {
	// IsEnabled returns true if the given flag is active for the given userID.
	// An empty userID evaluates the flag without user context (global flag).
	IsEnabled(ctx context.Context, flag, userID string) (bool, error)
}
