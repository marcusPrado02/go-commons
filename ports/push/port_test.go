package push_test

import (
	"context"
	"testing"

	"github.com/marcusPrado02/go-commons/ports/push"
)

// Compile-time check that Port can be implemented.
var _ push.Port = (*nilPush)(nil)

type nilPush struct{}

func (n *nilPush) Send(_ context.Context, _ push.Notification) (push.Receipt, error) {
	return push.Receipt{}, nil
}
func (n *nilPush) Ping(_ context.Context) error { return nil }

func TestPushNotification_Fields(t *testing.T) {
	n := push.Notification{
		Token: "tok-abc",
		Title: "Hello",
		Body:  "World",
		Data:  map[string]string{"key": "val"},
		Topic: "news",
	}
	if n.Token != "tok-abc" {
		t.Errorf("unexpected Token: %q", n.Token)
	}
	if n.Title != "Hello" {
		t.Errorf("unexpected Title: %q", n.Title)
	}
	if n.Body != "World" {
		t.Errorf("unexpected Body: %q", n.Body)
	}
	if n.Data["key"] != "val" {
		t.Errorf("unexpected Data: %v", n.Data)
	}
	if n.Topic != "news" {
		t.Errorf("unexpected Topic: %q", n.Topic)
	}
}

func TestPushReceipt_ZeroValue(t *testing.T) {
	var r push.Receipt
	if r.MessageID != "" {
		t.Fatalf("expected empty MessageID, got %q", r.MessageID)
	}
}
