// Package eventbus defines the port interface for in-process or distributed event publishing.
// Unlike ports/queue (delivery-oriented, at-least-once), EventBus is topic-based and
// optimised for fan-out broadcast within a single service or across services.
package eventbus

import "context"

// Event is a unit of information published on a topic.
type Event struct {
	// ID is a globally unique event identifier (e.g. UUID).
	ID string
	// Topic is the subject on which the event is published.
	Topic string
	// Payload is the serialised event body (JSON recommended).
	Payload []byte
	// Metadata carries optional key-value annotations.
	Metadata map[string]string
}

// Handler processes a received event. Returning an error does NOT trigger redelivery —
// use ports/queue for at-least-once delivery semantics.
type Handler func(ctx context.Context, event Event) error

// EventBus supports topic-based publish/subscribe.
type EventBus interface {
	// Publish broadcasts an event to all subscribers of the given topic.
	Publish(ctx context.Context, topic string, event Event) error
	// Subscribe registers a handler for events on the given topic.
	// The returned cancel function unregisters the handler.
	Subscribe(ctx context.Context, topic string, handler Handler) (cancel func(), err error)
}
