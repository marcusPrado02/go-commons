// Package ddd provides Domain-Driven Design primitives for go-commons.
// AggregateRoot is designed to be embedded — not extended via inheritance.
//
// Example usage:
//
//	type Order struct {
//	    ddd.AggregateRoot[OrderID]
//	    status OrderStatus
//	}
//
//	func PlaceOrder(id OrderID) *Order {
//	    o := &Order{AggregateRoot: ddd.NewAggregateRoot(id)}
//	    o.RegisterEvent(OrderPlaced{OccurredAt: time.Now()})
//	    return o
//	}
package ddd

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	OccurredAt() time.Time
	EventType() string
}

// AggregateRoot holds the aggregate's identity and pending domain events.
// Embed it in your aggregate struct — never inherit from it.
type AggregateRoot[ID any] struct {
	id     ID
	events []DomainEvent
}

// NewAggregateRoot creates an AggregateRoot with the given identifier.
func NewAggregateRoot[ID any](id ID) AggregateRoot[ID] {
	return AggregateRoot[ID]{id: id}
}

// ID returns the aggregate's identifier.
func (a *AggregateRoot[ID]) ID() ID { return a.id }

// RegisterEvent appends a domain event to the aggregate's pending event list.
// Events are not published until PullDomainEvents is called.
func (a *AggregateRoot[ID]) RegisterEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

// PullDomainEvents returns a copy of all pending domain events and clears the list.
// Safe to call multiple times — subsequent calls return empty slices until new events are registered.
func (a *AggregateRoot[ID]) PullDomainEvents() []DomainEvent {
	events := append([]DomainEvent(nil), a.events...)
	a.events = nil
	return events
}
