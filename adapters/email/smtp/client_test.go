package smtp_test

import (
	"context"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/email/smtp"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

func TestNew_CreatesClient(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	c := smtp.New("mail.example.com", 465, "user", "pass", from)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestSend_InvalidEmail_ReturnsValidationError(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	c := smtp.New("mail.example.com", 465, "user", "pass", from)

	// Email with no recipients fails Validate() before any network call.
	_, err := c.Send(context.Background(), emailport.Email{
		From:    from,
		Subject: "Hello",
		Text:    "World",
		// To is empty — Validate() must reject this.
	})
	if err == nil {
		t.Fatal("expected validation error for missing recipients")
	}
}

func TestSend_EmptyFromAndTo_ReturnsValidationError(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	c := smtp.New("mail.example.com", 465, "user", "pass", from)

	_, err := c.Send(context.Background(), emailport.Email{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSendWithTemplate_ReturnsUnsupportedError(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	to, _ := emailport.NewEmailAddress("to@example.com")
	c := smtp.New("mail.example.com", 465, "user", "pass", from)

	_, err := c.SendWithTemplate(context.Background(), emailport.TemplateEmailRequest{
		From: from,
		To:   []emailport.EmailAddress{to},
	})
	if err == nil {
		t.Fatal("expected unsupported error from SendWithTemplate")
	}
}

func TestPing_Unreachable_ReturnsError(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	// Port 1 is reserved and should never have an SMTP server.
	c := smtp.New("127.0.0.1", 1, "user", "pass", from)

	if err := c.Ping(context.Background()); err == nil {
		t.Fatal("expected error pinging unreachable server")
	}
}
