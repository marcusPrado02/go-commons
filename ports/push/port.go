// Package push defines the port interface for push notification delivery.
package push

import "context"

// PushPort delivers push notifications to mobile devices or browsers.
type PushPort interface {
	// Send delivers a push notification.
	Send(ctx context.Context, notification PushNotification) (PushReceipt, error)
	// Ping verifies connectivity and credential validity.
	Ping(ctx context.Context) error
}

// PushNotification describes a push message to be delivered.
type PushNotification struct {
	// Token is the device/browser registration token.
	Token string
	Title string
	Body  string
	Data  map[string]string
	// Topic is an optional topic for fan-out delivery (provider-specific).
	Topic string
}

// PushReceipt is returned after successful delivery.
type PushReceipt struct {
	MessageID string
}
