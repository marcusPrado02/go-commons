// Package assert provides fluent assertion helpers for domain objects.
package assert

import (
	"testing"

	"github.com/marcusPrado02/go-commons/kernel/ddd"
)

// Eventful is the structural constraint for aggregates that expose domain events.
// Any struct with PullDomainEvents() satisfies this — no embedding of AggregateRoot required.
type Eventful interface {
	PullDomainEvents() []ddd.DomainEvent
}

// AggregateAssertion provides fluent assertions on domain aggregates.
type AggregateAssertion[T Eventful] struct {
	t      testing.TB
	actual T
	events []ddd.DomainEvent
}

// Aggregate begins a fluent assertion chain on the given aggregate.
// PullDomainEvents is called once and the result is held for all subsequent assertions.
func Aggregate[T Eventful](t testing.TB, actual T) *AggregateAssertion[T] {
	t.Helper()
	return &AggregateAssertion[T]{
		t:      t,
		actual: actual,
		events: actual.PullDomainEvents(),
	}
}

// HasDomainEvents asserts that exactly count events were raised.
func (a *AggregateAssertion[T]) HasDomainEvents(count int) *AggregateAssertion[T] {
	a.t.Helper()
	if len(a.events) != count {
		a.t.Errorf("expected %d domain events, got %d", count, len(a.events))
	}
	return a
}

// HasNoDomainEvents asserts that no events were raised.
func (a *AggregateAssertion[T]) HasNoDomainEvents() *AggregateAssertion[T] {
	return a.HasDomainEvents(0)
}

// HasEventOfType asserts that at least one event has the given EventType().
func (a *AggregateAssertion[T]) HasEventOfType(eventType string) *AggregateAssertion[T] {
	a.t.Helper()
	for _, e := range a.events {
		if e.EventType() == eventType {
			return a
		}
	}
	a.t.Errorf("no domain event of type %q found among %d events", eventType, len(a.events))
	return a
}

// FirstEventSatisfies asserts that the first event satisfies the predicate.
func (a *AggregateAssertion[T]) FirstEventSatisfies(fn func(ddd.DomainEvent) bool) *AggregateAssertion[T] {
	a.t.Helper()
	if len(a.events) == 0 {
		a.t.Error("no domain events to assert on")
		return a
	}
	if !fn(a.events[0]) {
		a.t.Errorf("first domain event of type %q did not satisfy predicate", a.events[0].EventType())
	}
	return a
}
