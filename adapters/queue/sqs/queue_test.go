package sqs_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	sqsadapter "github.com/marcusPrado02/go-commons/adapters/queue/sqs"
	"github.com/marcusPrado02/go-commons/ports/queue"
)

// stubSQS is a minimal HTTP stub that responds to SQS XML requests.
type stubSQS struct {
	sendCalled  atomic.Int32
	deleteCalled atomic.Int32
	listCalled  atomic.Int32
	// messageBody is returned in ReceiveMessage responses.
	messageBody string
	receiptHandle string
}

func (s *stubSQS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	action := r.FormValue("Action")
	switch action {
	case "SendMessage":
		s.sendCalled.Add(1)
		fmt.Fprintf(w, `<SendMessageResponse><SendMessageResult><MessageId>msg-1</MessageId></SendMessageResult></SendMessageResponse>`)
	case "ReceiveMessage":
		if s.messageBody != "" {
			body := s.messageBody
			s.messageBody = "" // deliver once
			fmt.Fprintf(w, `<ReceiveMessageResponse><ReceiveMessageResult><Message><MessageId>msg-1</MessageId><ReceiptHandle>%s</ReceiptHandle><Body>%s</Body></Message></ReceiveMessageResult></ReceiveMessageResponse>`,
				xmlEscape(s.receiptHandle), xmlEscape(body))
		} else {
			fmt.Fprintf(w, `<ReceiveMessageResponse><ReceiveMessageResult></ReceiveMessageResult></ReceiveMessageResponse>`)
		}
	case "DeleteMessage":
		s.deleteCalled.Add(1)
		fmt.Fprintf(w, `<DeleteMessageResponse><ResponseMetadata><RequestId>x</RequestId></ResponseMetadata></DeleteMessageResponse>`)
	case "ListQueues":
		s.listCalled.Add(1)
		fmt.Fprintf(w, `<ListQueuesResponse><ListQueuesResult></ListQueuesResult></ListQueuesResponse>`)
	default:
		http.Error(w, "unknown action: "+action, http.StatusBadRequest)
	}
}

func xmlEscape(s string) string {
	b, _ := xml.Marshal(struct{ V string }{s})
	// xml.Marshal wraps in <V>…</V>; extract inner text
	raw := string(b)
	if len(raw) > 7 {
		return raw[3 : len(raw)-4]
	}
	return s
}

func newTestClient(t *testing.T, stub http.Handler) (*sqsadapter.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(stub)
	t.Cleanup(srv.Close)

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	sqsClient := awssqs.NewFromConfig(cfg, func(o *awssqs.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
	})
	return sqsadapter.New(sqsClient), srv
}

func TestSQS_Publish(t *testing.T) {
	stub := &stubSQS{}
	client, srv := newTestClient(t, stub)

	err := client.Publish(context.Background(), srv.URL+"/123456789/test-queue", queue.Message{
		Payload: []byte(`{"key":"value"}`),
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if stub.sendCalled.Load() != 1 {
		t.Errorf("expected SendMessage called once, got %d", stub.sendCalled.Load())
	}
}

func TestSQS_Ping(t *testing.T) {
	stub := &stubSQS{}
	client, srv := newTestClient(t, stub)

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if stub.listCalled.Load() != 1 {
		t.Errorf("expected ListQueues called once, got %d", stub.listCalled.Load())
	}
	_ = srv
}

func TestSQS_Subscribe_ReceivesAndDeletes(t *testing.T) {
	stub := &stubSQS{
		messageBody:   `hello sqs`,
		receiptHandle: "handle-abc",
	}
	client, srv := newTestClient(t, stub)

	received := make(chan string, 1)
	cancel, err := client.Subscribe(context.Background(), srv.URL+"/123/q", func(_ context.Context, msg queue.Message) error {
		received <- string(msg.Payload)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer cancel()

	select {
	case body := <-received:
		if body != "hello sqs" {
			t.Errorf("expected %q, got %q", "hello sqs", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	// Give the delete goroutine a moment to execute.
	time.Sleep(50 * time.Millisecond)
	if stub.deleteCalled.Load() == 0 {
		t.Error("expected DeleteMessage to be called after successful handler")
	}
}
