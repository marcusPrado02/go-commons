// Package fcm provides a PushPort implementation backed by Firebase Cloud Messaging.
package fcm

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/marcusPrado02/go-commons/ports/push"
)

// Client implements push.PushPort using Firebase Cloud Messaging.
type Client struct {
	msg *messaging.Client
}

// New creates a Client using a service account credentials file.
// credentialsFile is the path to a Firebase service account JSON file.
func New(ctx context.Context, credentialsFile string) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("fcm: init firebase app: %w", err)
	}
	return newFromApp(ctx, app)
}

// NewFromCredentialsJSON creates a Client from a credentials JSON byte slice.
// Useful for passing credentials from a secrets manager rather than from the filesystem.
func NewFromCredentialsJSON(ctx context.Context, credJSON []byte) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON(credJSON))
	if err != nil {
		return nil, fmt.Errorf("fcm: init firebase app: %w", err)
	}
	return newFromApp(ctx, app)
}

func newFromApp(ctx context.Context, app *firebase.App) (*Client, error) {
	msg, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: create messaging client: %w", err)
	}
	return &Client{msg: msg}, nil
}

// Send delivers a push notification via FCM.
// If notification.Token is set, the message is sent to a single device.
// If notification.Topic is set and Token is empty, the message is sent to the topic.
func (c *Client) Send(ctx context.Context, notification push.PushNotification) (push.PushReceipt, error) {
	msg, err := buildMessage(notification)
	if err != nil {
		return push.PushReceipt{}, err
	}

	msgID, err := c.msg.Send(ctx, msg)
	if err != nil {
		return push.PushReceipt{}, fmt.Errorf("fcm: send: %w", err)
	}
	return push.PushReceipt{MessageID: msgID}, nil
}

// Ping verifies FCM credentials are valid by performing a dry-run send.
// It sends a minimal message with validate_only=true; no message is actually delivered.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.msg.SendEachForMulticast(ctx, &messaging.MulticastMessage{
		Tokens: []string{"ping-token"},
		Notification: &messaging.Notification{
			Title: "ping",
		},
	})
	// FCM returns an error with INVALID_ARGUMENT for the fake token, but a
	// successful HTTP round-trip means credentials are valid. We interpret
	// any non-authentication error as a successful ping.
	if err != nil && isAuthError(err) {
		return fmt.Errorf("fcm: ping: %w", err)
	}
	return nil
}

func buildMessage(n push.PushNotification) (*messaging.Message, error) {
	if n.Token == "" && n.Topic == "" {
		return nil, fmt.Errorf("fcm: notification must have a Token or a Topic")
	}

	msg := &messaging.Message{
		Notification: &messaging.Notification{
			Title: n.Title,
			Body:  n.Body,
		},
		Data: n.Data,
	}
	if n.Token != "" {
		msg.Token = n.Token
	} else {
		msg.Topic = n.Topic
	}
	return msg, nil
}

// isAuthError returns true if err signals an authentication/authorization failure.
func isAuthError(err error) bool {
	msg := err.Error()
	for _, needle := range []string{"UNAUTHENTICATED", "PERMISSION_DENIED", "invalid_grant", "unauthorized"} {
		for i := 0; i+len(needle) <= len(msg); i++ {
			if msg[i:i+len(needle)] == needle {
				return true
			}
		}
	}
	return false
}

var _ push.PushPort = (*Client)(nil)
