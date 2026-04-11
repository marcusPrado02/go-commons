// Package push defines the port interface for push notification delivery.
package push

import "context"

// Port delivers push notifications to mobile devices or browsers.
type Port interface {
	// Send delivers a push notification.
	Send(ctx context.Context, notification Notification) (Receipt, error)
	// Ping verifies connectivity and credential validity.
	Ping(ctx context.Context) error
}

// Notification describes a push message to be delivered.
type Notification struct {
	// Token is the device/browser registration token.
	Token string
	Title string
	Body  string
	Data  map[string]string
	// Topic is an optional topic for fan-out delivery (provider-specific).
	Topic string
}

// Receipt is returned after successful delivery.
type Receipt struct {
	MessageID string
}
