// Package twilio provides a Twilio implementation of ports/sms.SMSPort.
package twilio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/marcusPrado02/go-commons/ports/sms"
	twilioapi "github.com/twilio/twilio-go"
	twilioClient "github.com/twilio/twilio-go/client"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Client is a Twilio implementation of SMSPort.
type Client struct {
	twilio     *twilioapi.RestClient
	from       string
	accountSID string
}

// Option configures a Twilio Client.
type Option func(*twilioapi.ClientParams)

// WithHTTPClient injects a custom HTTP client into the underlying Twilio SDK.
// Used in tests to intercept HTTP requests with a mock server.
// Credentials are carried from the New() arguments so the SDK can authenticate.
func WithHTTPClient(hc *http.Client) Option {
	return func(p *twilioapi.ClientParams) {
		creds := twilioClient.NewCredentials(p.Username, p.Password)
		p.Client = &twilioClient.Client{Credentials: creds, HTTPClient: hc}
	}
}

// New creates a new Twilio SMS client.
func New(accountSID, authToken, fromNumber string, opts ...Option) (*Client, error) {
	if accountSID == "" || authToken == "" || fromNumber == "" {
		return nil, fmt.Errorf("twilio: accountSID, authToken, and fromNumber are required")
	}
	params := twilioapi.ClientParams{
		Username: accountSID,
		Password: authToken,
	}
	for _, o := range opts {
		o(&params)
	}
	client := twilioapi.NewRestClientWithParams(params)
	return &Client{twilio: client, from: fromNumber, accountSID: accountSID}, nil
}

// Send delivers an SMS message to the given E.164 phone number.
func (c *Client) Send(_ context.Context, to, body string) (sms.SMSReceipt, error) {
	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(c.from)
	params.SetBody(body)

	msg, err := c.twilio.Api.CreateMessage(params)
	if err != nil {
		return sms.SMSReceipt{}, fmt.Errorf("twilio: send failed: %w", err)
	}
	if msg.Sid == nil {
		return sms.SMSReceipt{}, fmt.Errorf("twilio: no message SID returned")
	}
	return sms.SMSReceipt{MessageID: *msg.Sid}, nil
}

// Ping verifies Twilio credentials by fetching the account.
func (c *Client) Ping(_ context.Context) error {
	_, err := c.twilio.Api.FetchAccount(c.accountSID)
	if err != nil {
		return fmt.Errorf("twilio: ping failed: %w", err)
	}
	return nil
}

var _ sms.SMSPort = (*Client)(nil)
