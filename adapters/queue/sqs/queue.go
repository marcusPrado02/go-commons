// Package sqs provides a Port implementation backed by AWS SQS.
package sqs

import (
	"context"
	"fmt"
	"sync"

	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/marcusPrado02/go-commons/ports/queue"
)

const (
	defaultMaxMessages = 10
	defaultWaitSeconds = 20 // long-polling
)

// Client implements queue.Port using AWS SQS.
// Topic names are used as SQS queue URLs directly.
type Client struct {
	sqs *awssqs.Client
	mu  sync.Mutex
}

// New creates a Client from an existing *awssqs.Client.
// Obtain one via awssqs.NewFromConfig(cfg) with the desired region/credentials.
func New(client *awssqs.Client) *Client {
	return &Client{sqs: client}
}

// Publish sends msg to the SQS queue identified by topic (queue URL).
func (c *Client) Publish(ctx context.Context, topic string, msg queue.Message) error {
	body := string(msg.Payload)
	attrs := make(map[string]types.MessageAttributeValue, len(msg.Attributes))
	for k, v := range msg.Attributes {
		v := v
		attrs[k] = types.MessageAttributeValue{
			DataType:    strPtr("String"),
			StringValue: &v,
		}
	}

	_, err := c.sqs.SendMessage(ctx, &awssqs.SendMessageInput{
		QueueUrl:          &topic,
		MessageBody:       &body,
		MessageAttributes: attrs,
	})
	if err != nil {
		return fmt.Errorf("sqs: publish to %q: %w", topic, err)
	}
	return nil
}

// Subscribe starts a long-polling goroutine that delivers messages from the SQS queue
// identified by topic (queue URL) to handler. Returns a cancel func that stops polling.
// Messages where handler returns nil are deleted; errors leave them for redelivery.
func (c *Client) Subscribe(ctx context.Context, topic string, handler queue.Handler) (func(), error) {
	pCtx, cancel := context.WithCancel(ctx)
	go c.poll(pCtx, topic, handler)
	return cancel, nil
}

// Ping checks that the SQS endpoint is reachable by listing queue attributes.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.sqs.ListQueues(ctx, &awssqs.ListQueuesInput{MaxResults: int32Ptr(1)})
	if err != nil {
		return fmt.Errorf("sqs: ping: %w", err)
	}
	return nil
}

// poll runs a long-polling receive loop until ctx is cancelled.
func (c *Client) poll(ctx context.Context, queueURL string, handler queue.Handler) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		out, err := c.sqs.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{
			QueueUrl:              &queueURL,
			MaxNumberOfMessages:   defaultMaxMessages,
			WaitTimeSeconds:       defaultWaitSeconds,
			MessageAttributeNames: []string{"All"},
		})
		if err != nil {
			// Context cancelled — normal shutdown.
			return
		}

		for _, m := range out.Messages {
			msg := toQueueMessage(m)
			if handler(ctx, msg) == nil && m.ReceiptHandle != nil {
				_, _ = c.sqs.DeleteMessage(ctx, &awssqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: m.ReceiptHandle,
				})
			}
		}
	}
}

func toQueueMessage(m types.Message) queue.Message {
	payload := []byte{}
	if m.Body != nil {
		payload = []byte(*m.Body)
	}
	id := ""
	if m.MessageId != nil {
		id = *m.MessageId
	}
	attrs := make(map[string]string, len(m.MessageAttributes))
	for k, v := range m.MessageAttributes {
		if v.StringValue != nil {
			attrs[k] = *v.StringValue
		}
	}
	return queue.Message{ID: id, Payload: payload, Attributes: attrs}
}

func strPtr(s string) *string     { return &s }
func int32Ptr(n int32) *int32     { return &n }

var _ queue.Port = (*Client)(nil)
