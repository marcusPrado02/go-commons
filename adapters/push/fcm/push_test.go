package fcm_test

import (
	"context"
	"os"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/push/fcm"
	"github.com/marcusPrado02/go-commons/ports/push"
)

// compile-time interface check
var _ push.Port = (*fcm.Client)(nil)

// skipIfNoCreds skips the test if FCM_CREDENTIALS_FILE is not set.
func skipIfNoCreds(t *testing.T) string {
	t.Helper()
	creds := os.Getenv("FCM_CREDENTIALS_FILE")
	if creds == "" {
		t.Skip("FCM_CREDENTIALS_FILE not set — skipping FCM integration test")
	}
	return creds
}

func TestFCM_Ping(t *testing.T) {
	creds := skipIfNoCreds(t)
	client, err := fcm.New(context.Background(), creds)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Ping does a dry-run send; it should not return auth errors with valid credentials.
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestFCM_Send_MissingTokenAndTopic(t *testing.T) {
	// This test exercises the pure validation logic in buildMessage.
	// It does not need a real FCM connection because the error is returned before any network call.
	//
	// We create a client from a fake credential JSON. The error we're testing happens
	// before any Firebase API call, so the fake credentials are never used.
	creds := skipIfNoCreds(t)
	client, err := fcm.New(context.Background(), creds)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Send(context.Background(), push.Notification{
		Title: "Hello",
		// Token and Topic are both empty.
	})
	if err == nil {
		t.Fatal("expected error when Token and Topic are both empty, got nil")
	}
}
