// Package queue defines the port interface for message queue operations.
package queue

import "context"

// Message is a unit of work published to or received from a queue.
type Message struct {
	ID    string
	Topic string
	// Payload is the serialized message body. Use JSON or protobuf encoding.
	Payload []byte
	// Attributes are optional key-value metadata attached to the message.
	Attributes map[string]string
}

// Handler processes a received message. Return an error to trigger redelivery.
type Handler func(ctx context.Context, msg Message) error

// QueuePort supports publishing and subscribing to topics.
type QueuePort interface {
	// Publish sends a message to the given topic.
	Publish(ctx context.Context, topic string, msg Message) error
	// Subscribe registers a handler for messages on the given topic.
	// The returned cancel function unregisters the handler.
	Subscribe(ctx context.Context, topic string, handler Handler) (cancel func(), err error)
	// Ping verifies connectivity to the queue broker.
	Ping(ctx context.Context) error
}
