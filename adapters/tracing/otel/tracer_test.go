package otel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/tracing/otel"
	obs "github.com/marcusPrado02/go-commons/ports/observability"
	otelnoop "go.opentelemetry.io/otel/trace/noop"
)

func newTracer() *otel.Tracer {
	return otel.New(otelnoop.NewTracerProvider().Tracer("test"))
}

func TestStartSpan_ReturnsNonNilSpanAndContext(t *testing.T) {
	tracer := newTracer()
	ctx, span := tracer.StartSpan(context.Background(), "test-span")
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	span.End()
}

func TestStartSpan_ImplementsTracerInterface(t *testing.T) {
	var _ obs.Tracer = newTracer()
}

func TestSpan_SetAttribute_AllSupportedTypes(t *testing.T) {
	_, span := newTracer().StartSpan(context.Background(), "attr-test")
	defer span.End()

	// Must not panic for any supported type.
	span.SetAttribute("str", "value")
	span.SetAttribute("int", 42)
	span.SetAttribute("int64", int64(99))
	span.SetAttribute("float64", 3.14)
	span.SetAttribute("bool", true)
	span.SetAttribute("other", []byte("raw")) // falls through to Sprintf
}

func TestSpan_RecordError_DoesNotPanic(t *testing.T) {
	_, span := newTracer().StartSpan(context.Background(), "err-span")
	defer span.End()

	// RecordError must not panic and must set the error status.
	span.RecordError(errors.New("something went wrong"))
}

func TestSpan_End_IsIdempotent(t *testing.T) {
	_, span := newTracer().StartSpan(context.Background(), "idempotent")
	// Calling End twice must not panic (noop provider is safe).
	span.End()
	span.End()
}
