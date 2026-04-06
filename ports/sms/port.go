// Package sms defines the port interface for SMS delivery.
package sms

import "context"

// SMSPort sends SMS messages via a configured provider.
type SMSPort interface {
	// Send delivers a text message to the given E.164 phone number.
	Send(ctx context.Context, to, body string) (SMSReceipt, error)
	// Ping verifies connectivity and credential validity.
	Ping(ctx context.Context) error
}

// SMSReceipt is returned by the provider after successful delivery.
type SMSReceipt struct {
	MessageID string
}
