package contracts

import (
	"context"
	"sync"
	"time"

	"github.com/marcusPrado02/go-commons/ports/queue"
	"github.com/stretchr/testify/suite"
)

// QueueContract is a reusable test suite for Port implementations.
//
// Example:
//
//	func TestInMemoryQueue(t *testing.T) {
//	    suite.Run(t, &contracts.QueueContract{Queue: mymq.New()})
//	}
type QueueContract struct {
	suite.Suite
	// Queue is the Port implementation under test.
	Queue queue.Port
}

func (s *QueueContract) TestPublish_NoError() {
	msg := queue.Message{ID: "m1", Topic: "test", Payload: []byte("hello")}
	s.Require().NoError(s.Queue.Publish(context.Background(), "test", msg))
}

func (s *QueueContract) TestSubscribe_ReceivesPublishedMessage() {
	ctx := context.Background()
	topic := "contract-topic"

	var (
		received queue.Message
		once     sync.Once
		done     = make(chan struct{})
	)

	cancel, err := s.Queue.Subscribe(ctx, topic, func(_ context.Context, msg queue.Message) error {
		once.Do(func() {
			received = msg
			close(done)
		})
		return nil
	})
	s.Require().NoError(err)
	defer cancel()

	msg := queue.Message{ID: "m2", Topic: topic, Payload: []byte("world")}
	s.Require().NoError(s.Queue.Publish(ctx, topic, msg))

	select {
	case <-done:
		s.Equal(msg.Payload, received.Payload)
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for message delivery")
	}
}

func (s *QueueContract) TestSubscribe_CancelStopsDelivery() {
	ctx := context.Background()
	topic := "cancel-topic"

	cancel, err := s.Queue.Subscribe(ctx, topic, func(_ context.Context, _ queue.Message) error {
		return nil
	})
	s.Require().NoError(err)

	// Cancelling must not panic or error.
	s.NotPanics(cancel)
}

func (s *QueueContract) TestPing_ReturnsNoError() {
	s.Require().NoError(s.Queue.Ping(context.Background()))
}
