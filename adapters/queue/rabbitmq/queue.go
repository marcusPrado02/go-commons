// Package rabbitmq provides a QueuePort implementation backed by RabbitMQ via amqp091-go.
package rabbitmq

import (
	"context"
	"fmt"
	"sync/atomic"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/marcusPrado02/go-commons/ports/queue"
)

// Client implements queue.QueuePort using RabbitMQ.
//
// Topics map to AMQP queue names. Publish uses the default exchange ("") with
// the topic as the routing key, which delivers directly to the named queue.
// Subscribe declares the queue as durable so it survives broker restarts.
type Client struct {
	conn    *amqp.Connection
	tagSeq  atomic.Uint64
}

// New creates a Client from an existing *amqp.Connection.
// Obtain one via amqp.Dial("amqp://guest:guest@localhost:5672/").
func New(conn *amqp.Connection) *Client {
	return &Client{conn: conn}
}

// Publish sends msg to the queue identified by topic.
// The queue is declared as durable before publishing; it is a no-op if it already exists.
func (c *Client) Publish(ctx context.Context, topic string, msg queue.Message) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer ch.Close()

	if err := declareQueue(ch, topic); err != nil {
		return err
	}

	headers := amqp.Table{}
	for k, v := range msg.Attributes {
		headers[k] = v
	}

	pub := amqp.Publishing{
		ContentType:  "application/octet-stream",
		Body:         msg.Payload,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
	}
	if msg.ID != "" {
		pub.MessageId = msg.ID
	}

	if err := ch.PublishWithContext(ctx, "", topic, false, false, pub); err != nil {
		return fmt.Errorf("rabbitmq: publish to %q: %w", topic, err)
	}
	return nil
}

// Subscribe registers handler for messages arriving on the given topic (queue name).
// The queue is declared as durable. Messages where handler returns nil are ack'd;
// errors cause a nack with requeue=true (RabbitMQ will redeliver).
// The returned cancel func stops consumption and closes the channel.
func (c *Client) Subscribe(ctx context.Context, topic string, handler queue.Handler) (func(), error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: open channel: %w", err)
	}

	if err := declareQueue(ch, topic); err != nil {
		ch.Close()
		return nil, err
	}

	tag := fmt.Sprintf("go-commons-%d", c.tagSeq.Add(1))
	deliveries, err := ch.Consume(topic, tag, false, false, false, false, nil)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("rabbitmq: consume %q: %w", topic, err)
	}

	pCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer ch.Close()
		for {
			select {
			case <-pCtx.Done():
				_ = ch.Cancel(tag, false)
				return
			case d, ok := <-deliveries:
				if !ok {
					return
				}
				msg := toQueueMessage(d)
				if handler(pCtx, msg) == nil {
					_ = d.Ack(false)
				} else {
					_ = d.Nack(false, true) // requeue
				}
			}
		}
	}()

	return cancel, nil
}

// Ping verifies the connection is alive by checking its IsClosed state
// and attempting to open and immediately close a channel.
func (c *Client) Ping(_ context.Context) error {
	if c.conn.IsClosed() {
		return fmt.Errorf("rabbitmq: connection is closed")
	}
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: ping: %w", err)
	}
	return ch.Close()
}

// declareQueue declares a durable, non-exclusive queue with no extra arguments.
// It is idempotent — safe to call on every Publish/Subscribe.
func declareQueue(ch *amqp.Channel, name string) error {
	_, err := ch.QueueDeclare(name, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq: declare queue %q: %w", name, err)
	}
	return nil
}

func toQueueMessage(d amqp.Delivery) queue.Message {
	attrs := make(map[string]string, len(d.Headers))
	for k, v := range d.Headers {
		if s, ok := v.(string); ok {
			attrs[k] = s
		}
	}
	return queue.Message{
		ID:         d.MessageId,
		Topic:      d.RoutingKey,
		Payload:    d.Body,
		Attributes: attrs,
	}
}

var _ queue.QueuePort = (*Client)(nil)
