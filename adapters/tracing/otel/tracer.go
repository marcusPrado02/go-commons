// Package otel implements ports/observability.Tracer using OpenTelemetry.
package otel

import (
	"context"
	"fmt"

	obs "github.com/marcusPrado02/go-commons/ports/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer is an OpenTelemetry implementation of ports/observability.Tracer.
type Tracer struct {
	tracer oteltrace.Tracer
}

// New creates a Tracer from an OpenTelemetry tracer.
func New(t oteltrace.Tracer) *Tracer {
	return &Tracer{tracer: t}
}

// StartSpan creates a new OTel span derived from ctx.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, obs.Span) {
	ctx, span := t.tracer.Start(ctx, name)
	return ctx, &otelSpan{span: span}
}

type otelSpan struct {
	span oteltrace.Span
}

// End marks the OTel span as complete.
func (s *otelSpan) End() { s.span.End() }

// RecordError attaches an error to the span.
func (s *otelSpan) RecordError(err error) {
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
}

// SetAttribute adds a key-value attribute to the span.
func (s *otelSpan) SetAttribute(key string, value any) {
	switch v := value.(type) {
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

var _ obs.Tracer = (*Tracer)(nil)
var _ obs.Span = (*otelSpan)(nil)
