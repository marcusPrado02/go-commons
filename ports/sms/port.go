// Package sms defines the port interface for SMS delivery.
package sms

import "context"

// Port sends SMS messages via a configured provider.
type Port interface {
	// Send delivers a text message to the given E.164 phone number.
	Send(ctx context.Context, to, body string) (Receipt, error)
	// Ping verifies connectivity and credential validity.
	Ping(ctx context.Context) error
}

// Receipt is returned by the provider after successful delivery.
type Receipt struct {
	MessageID string
}
