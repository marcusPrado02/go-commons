// Package sendgrid provides a SendGrid implementation of ports/email.EmailPort.
package sendgrid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

const defaultBaseURL = "https://api.sendgrid.com"

// RetryConfig controls automatic retry on 429 and 5xx responses.
// Use WithRetry to attach it to a Client.
type RetryConfig struct {
	// MaxAttempts is the number of retries after the initial attempt (0 = no retries).
	MaxAttempts int
	// BaseDelay is the initial retry delay.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff. Zero means no cap.
	MaxDelay time.Duration
}

// retryableError marks HTTP errors that should trigger a retry (429, 5xx).
type retryableError struct{ cause error }

func (e *retryableError) Error() string { return e.cause.Error() }
func (e *retryableError) Unwrap() error { return e.cause }

// Client is a SendGrid implementation of EmailPort.
type Client struct {
	apiKey  string
	from    emailport.EmailAddress
	baseURL string
	http    *http.Client
	retry   *RetryConfig
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

// WithRetry enables automatic retry on 429 and 5xx responses using exponential
// backoff with full jitter — the same strategy used in app/resilience.
func WithRetry(cfg RetryConfig) Option {
	return func(c *Client) { c.retry = &cfg }
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
// Retries on 429 and 5xx if WithRetry is configured.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: invalid email: %w", err)
	}

	body, err := c.buildPayload(email)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sendgrid: failed to build payload: %w", err)
	}

	var receipt emailport.EmailReceipt
	action := func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/mail/send", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("sendgrid: failed to create request: %w", err)
		}
		c.setHeaders(req)

		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("sendgrid: request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return &retryableError{cause: fmt.Errorf("sendgrid: retryable status %d", resp.StatusCode)}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
		}
		receipt = emailport.EmailReceipt{MessageID: resp.Header.Get("X-Message-Id")}
		return nil
	}

	if err := c.runWithRetry(ctx, action); err != nil {
		return emailport.EmailReceipt{}, err
	}
	return receipt, nil
}

// SendWithTemplate delivers a template-based email via the SendGrid v3 API.
// Retries on 429 and 5xx if WithRetry is configured.
func (c *Client) SendWithTemplate(ctx context.Context, req emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	tos := make([]map[string]string, len(req.To))
	for i, t := range req.To {
		tos[i] = map[string]string{"email": t.Value}
	}
	payload := map[string]any{
		"from":                  map[string]string{"email": req.From.Value},
		"template_id":           req.TemplateName,
		"dynamic_template_data": req.Variables,
		"personalizations":      []map[string]any{{"to": tos}},
	}
	body, _ := json.Marshal(payload)

	var receipt emailport.EmailReceipt
	action := func(ctx context.Context) error {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v3/mail/send", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("sendgrid: failed to create request: %w", err)
		}
		c.setHeaders(httpReq)

		resp, err := c.http.Do(httpReq)
		if err != nil {
			return fmt.Errorf("sendgrid: request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return &retryableError{cause: fmt.Errorf("sendgrid: retryable status %d", resp.StatusCode)}
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("sendgrid: unexpected status %d", resp.StatusCode)
		}
		receipt = emailport.EmailReceipt{MessageID: resp.Header.Get("X-Message-Id")}
		return nil
	}

	if err := c.runWithRetry(ctx, action); err != nil {
		return emailport.EmailReceipt{}, err
	}
	return receipt, nil
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

// runWithRetry executes action, retrying on *retryableError up to RetryConfig.MaxAttempts times.
// Non-retryable errors (e.g. 4xx) are returned immediately without retry.
func (c *Client) runWithRetry(ctx context.Context, action func(context.Context) error) error {
	maxAttempts := 0
	if c.retry != nil {
		maxAttempts = c.retry.MaxAttempts
	}
	var lastErr error
	for attempt := 0; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := action(ctx)
		if err == nil {
			return nil
		}
		var re *retryableError
		if !errors.As(err, &re) {
			return err // permanent failure — do not retry
		}
		lastErr = err
		if attempt < maxAttempts {
			delay := retryJitter(attempt, c.retry.BaseDelay, c.retry.MaxDelay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return lastErr
}

// retryJitter computes exponential backoff with full jitter, matching app/resilience strategy.
func retryJitter(attempt int, base, maxDelay time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	exp := time.Duration(math.Pow(2, float64(attempt))) * base
	cap := exp
	if maxDelay > 0 && exp > maxDelay {
		cap = maxDelay
	}
	//nolint:gosec // math/rand is fine for jitter
	return time.Duration(rand.Int63n(int64(cap) + 1))
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
