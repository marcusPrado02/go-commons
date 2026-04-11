// Package transaction defines the port interface for database transaction management.
package transaction

import "context"

// Manager provides database-agnostic transaction management.
// Implementations wrap the underlying database/ORM transaction primitives.
type Manager interface {
	// Begin starts a new transaction and returns a derived context that carries it.
	Begin(ctx context.Context) (context.Context, error)
	// Commit commits the transaction stored in ctx.
	Commit(ctx context.Context) error
	// Rollback rolls back the transaction stored in ctx.
	// Should be called in a defer after Begin to ensure cleanup on error paths.
	Rollback(ctx context.Context) error
	// WithTx executes fn inside a transaction, committing on success and rolling
	// back on error or panic. If fn returns an error, it is returned by WithTx.
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
