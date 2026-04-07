// Package audit defines the port interface for audit logging.
package audit

import (
	"context"
	"time"
)

// AuditLog records immutable audit events for compliance and forensics.
type AuditLog interface {
	// Record persists a single audit event. Implementations must be idempotent
	// on AuditEvent.ID to avoid duplicate entries on retry.
	Record(ctx context.Context, event AuditEvent) error
}

// AuditEvent describes a single auditable action.
type AuditEvent struct {
	// ID is a globally unique identifier for the event (e.g. UUID). Used for idempotency.
	ID string
	// ActorID identifies who performed the action (user ID, service account, etc.).
	ActorID string
	// Action is a human-readable verb describing what happened (e.g. "user.login").
	Action string
	// Resource identifies what was acted upon (e.g. "order/42").
	Resource string
	// OccurredAt is when the event occurred.
	OccurredAt time.Time
	// Metadata contains optional additional context (e.g. IP address, request ID).
	Metadata map[string]string
}
