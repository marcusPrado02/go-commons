package observability

import "context"

// Logger is the structured logging port. All methods accept a variadic list of Fields.
// Errors should always be passed as obs.Err(err) — never as a separate parameter.
type Logger interface {
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	// Error logs at error level. Pass the error via obs.Err(err) in the fields.
	Error(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)
}

// Counter tracks a monotonically increasing value.
type Counter interface {
	Inc()
	Add(v float64)
}

// Observer records observed values (e.g. durations, sizes).
type Observer interface {
	Observe(v float64)
}

// Metrics is the metrics port for counters and histograms.
// Labels use Field to avoid ordering bugs — label names and values are explicit.
type Metrics interface {
	Counter(name string, labels ...Field) Counter
	Histogram(name string, labels ...Field) Observer
}

// Span represents an active tracing span. Always call End() when the operation completes.
type Span interface {
	// End marks the span as complete. Must always be called (defer recommended).
	End()
	// RecordError attaches an error to the span. Aligned with OpenTelemetry API.
	RecordError(err error)
	// SetAttribute adds a key-value attribute to the span.
	SetAttribute(key string, value any)
}

// Tracer creates and manages tracing spans.
type Tracer interface {
	// StartSpan creates a new child span derived from ctx.
	// Always call span.End() when the operation is complete.
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}
