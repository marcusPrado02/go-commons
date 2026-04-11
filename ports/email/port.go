// Package email defines the port interface for email delivery.
package email

import (
	"context"
	"fmt"
	"net/mail"
)

// Port is the primary port for sending email messages.
type Port interface {
	// Send delivers a single email message.
	Send(ctx context.Context, email Email) (Receipt, error)
	// SendWithTemplate delivers an email using a named template.
	SendWithTemplate(ctx context.Context, req TemplateEmailRequest) (Receipt, error)
	// Ping verifies the email provider is reachable and credentials are valid.
	Ping(ctx context.Context) error
}

// Address is a validated email address value object.
// Always construct via NewEmailAddress — never create the struct directly.
type Address struct {
	Value string
}

// NewEmailAddress parses and validates an email address per RFC 5322.
func NewEmailAddress(value string) (Address, error) {
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return Address{}, fmt.Errorf("invalid email address %q: %w", value, err)
	}
	return Address{Value: addr.Address}, nil
}

// Email represents a composed email message ready for delivery.
type Email struct {
	From    Address
	To      []Address
	CC      []Address
	BCC     []Address
	Subject string
	// HTML is the HTML body of the email. At least one of HTML or Text must be set.
	HTML string
	// Text is the plain-text body. At least one of HTML or Text must be set.
	Text    string
	ReplyTo *Address
}

// Validate checks that the email satisfies minimum delivery requirements.
//
//nolint:gocritic // hugeParam: Email is a domain value object; changing to pointer receiver is an API break.
func (e Email) Validate() error {
	if len(e.To) == 0 {
		return fmt.Errorf("email must have at least one recipient")
	}
	if e.HTML == "" && e.Text == "" {
		return fmt.Errorf("email must have an HTML or text body")
	}
	if e.From.Value == "" {
		return fmt.Errorf("email must have a From address")
	}
	return nil
}

// Receipt is returned by the provider after successful delivery.
type Receipt struct {
	// MessageID is the provider-assigned message identifier.
	MessageID string
}

// TemplateEmailRequest requests delivery of a pre-defined template.
type TemplateEmailRequest struct {
	From         Address
	To           []Address
	TemplateName string
	Variables    map[string]any
}
