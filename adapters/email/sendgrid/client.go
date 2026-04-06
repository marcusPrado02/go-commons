// Package sendgrid provides a SendGrid implementation of ports/email.EmailPort.
package sendgrid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

const defaultBaseURL = "https://api.sendgrid.com"

// Client is a SendGrid implementation of EmailPort.
type Client struct {
	apiKey  string
	from    emailport.EmailAddress
	baseURL string
	http    *http.Client
}

// Option configures a SendGrid Client.
type Option func(*Client)

// WithBaseURL overrides the SendGrid API base URL. Used for testing with a mock server.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.Timeout = d }
}

// New creates a new SendGrid client.
func New(apiKey string, from emailport.EmailAddress, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("sendgrid: apiKey cannot be empty")
	}
	c := &Client{
		apiKey:  apiKey,
		from:    from,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Send delivers an email via the SendGrid v3 Mail Send API.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: invalid email: %w", err)
	}

	body, err := c.buildPayload(email)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: failed to build payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: failed to create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
	}

	return emailport.EmailReceipt{MessageID: resp.Header.Get("X-Message-Id")}, nil
}

// SendWithTemplate delivers a template-based email via the SendGrid v3 API.
func (c *Client) SendWithTemplate(ctx context.Context, req emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	tos := make([]map[string]string, len(req.To))
	for i, t := range req.To {
		tos[i] = map[string]string{"email": t.Value}
	}
	payload := map[string]any{
		"from":                     map[string]string{"email": req.From.Value},
		"template_id":              req.TemplateName,
		"dynamic_template_data":    req.Variables,
		"personalizations":         []map[string]any{{"to": tos}},
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return emailport.EmailReceipt{}, err
	}
	c.setHeaders(httpReq)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return emailport.EmailReceipt{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
	}
	return emailport.EmailReceipt{MessageID: resp.Header.Get("X-Message-Id")}, nil
}

// Ping verifies SendGrid connectivity by calling the mail settings endpoint.
func (c *Client) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(pingCtx, http.MethodGet, c.baseURL+"/v3/mail/settings", nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sendgrid: ping returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

func (c *Client) buildPayload(email emailport.Email) ([]byte, error) {
	tos := make([]map[string]string, len(email.To))
	for i, t := range email.To {
		tos[i] = map[string]string{"email": t.Value}
	}

	content := []map[string]string{}
	if email.HTML != "" {
		content = append(content, map[string]string{"type": "text/html", "value": email.HTML})
	}
	if email.Text != "" {
		content = append(content, map[string]string{"type": "text/plain", "value": email.Text})
	}

	payload := map[string]any{
		"personalizations": []map[string]any{{"to": tos, "subject": email.Subject}},
		"from":             map[string]string{"email": email.From.Value},
		"content":          content,
	}

	return json.Marshal(payload)
}

var _ emailport.EmailPort = (*Client)(nil)
