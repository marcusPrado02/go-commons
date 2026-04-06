// Package email defines the port interface for email delivery.
package email

import (
	"context"
	"fmt"
	"net/mail"
)

// EmailPort is the primary port for sending email messages.
type EmailPort interface {
	// Send delivers a single email message.
	Send(ctx context.Context, email Email) (EmailReceipt, error)
	// SendWithTemplate delivers an email using a named template.
	SendWithTemplate(ctx context.Context, req TemplateEmailRequest) (EmailReceipt, error)
	// Ping verifies the email provider is reachable and credentials are valid.
	Ping(ctx context.Context) error
}

// EmailAddress is a validated email address value object.
// Always construct via NewEmailAddress — never create the struct directly.
type EmailAddress struct {
	Value string
}

// NewEmailAddress parses and validates an email address per RFC 5322.
func NewEmailAddress(value string) (EmailAddress, error) {
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return EmailAddress{}, fmt.Errorf("invalid email address %q: %w", value, err)
	}
	return EmailAddress{Value: addr.Address}, nil
}

// Email represents a composed email message ready for delivery.
type Email struct {
	From    EmailAddress
	To      []EmailAddress
	CC      []EmailAddress
	BCC     []EmailAddress
	Subject string
	// HTML is the HTML body of the email. At least one of HTML or Text must be set.
	HTML string
	// Text is the plain-text body. At least one of HTML or Text must be set.
	Text    string
	ReplyTo *EmailAddress
}

// Validate checks that the email satisfies minimum delivery requirements.
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

// EmailReceipt is returned by the provider after successful delivery.
type EmailReceipt struct {
	// MessageID is the provider-assigned message identifier.
	MessageID string
}

// TemplateEmailRequest requests delivery of a pre-defined template.
type TemplateEmailRequest struct {
	From         EmailAddress
	To           []EmailAddress
	TemplateName string
	Variables    map[string]any
}
