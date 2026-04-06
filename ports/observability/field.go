// Package observability defines port interfaces for logging, metrics, and tracing.
// These interfaces are dependency-free — adapters (slog, prometheus, otel) implement them.
package observability

// Field is a structured key-value pair used across logging and metrics APIs.
// Using the same type for both logging and metrics creates a consistent vocabulary.
type Field struct {
	Key   string
	Value any
}

// F creates a Field with the given key and value.
func F(key string, value any) Field { return Field{Key: key, Value: value} }

// Err creates a Field for error logging. Use instead of passing err as a separate parameter.
//
//	logger.Error(ctx, "failed to send email", obs.Err(err))
func Err(err error) Field { return F("error", err) }

// RequestID creates a Field for the current request identifier.
func RequestID(id string) Field { return F("request.id", id) }

// UserID creates a Field for the current user identifier.
func UserID(id string) Field { return F("user.id", id) }
