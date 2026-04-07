// Package smtp provides an SMTP implementation of ports/email.EmailPort using stdlib net/smtp.
package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"strings"
	"time"

	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

// Client is an SMTP implementation of EmailPort.
type Client struct {
	host     string
	port     int
	username string
	password string
	from     emailport.EmailAddress
	timeout  time.Duration
}

// Option configures an SMTP Client.
type Option func(*Client)

// WithTimeout sets the SMTP connection timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// New creates a new SMTP client.
func New(host string, port int, username, password string, from emailport.EmailAddress, opts ...Option) *Client {
	c := &Client{host: host, port: port, username: username, password: password, from: from, timeout: 30 * time.Second}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Send delivers an email via SMTP with TLS.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	auth := smtp.PlainAuth("", c.username, c.password, c.host)

	tos := make([]string, len(email.To))
	for i, t := range email.To {
		tos[i] = t.Value
	}

	msg, err := c.buildMessage(email, tos)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: build message failed: %w", err)
	}

	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: c.host})
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: dial failed: %w", err)
	}

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		_ = conn.Close()
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: client creation failed: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: auth failed: %w", err)
	}
	if err := client.Mail(c.from.Value); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: MAIL FROM failed: %w", err)
	}
	for _, to := range tos {
		if err := client.Rcpt(to); err != nil {
			return emailport.EmailReceipt{}, fmt.Errorf("smtp: RCPT TO %q failed: %w", to, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: DATA failed: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: write message failed: %w", err)
	}
	if err = w.Close(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("smtp: close DATA failed: %w", err)
	}
	return emailport.EmailReceipt{}, nil
}

// SendWithTemplate is not natively supported by SMTP — render template first, then call Send.
func (c *Client) SendWithTemplate(_ context.Context, _ emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	return emailport.EmailReceipt{}, fmt.Errorf("smtp: SendWithTemplate not supported — render template with TemplatePort first, then call Send")
}

// Ping verifies that the SMTP server is reachable by opening a TLS connection.
func (c *Client) Ping(_ context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: c.host})
	if err != nil {
		return fmt.Errorf("smtp: ping failed: %w", err)
	}
	return conn.Close()
}

// buildMessage constructs the RFC 5322 message bytes.
// tos must contain the resolved To addresses (already extracted from email.To).
func (c *Client) buildMessage(email emailport.Email, tos []string) ([]byte, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	buf.WriteString("From: " + c.from.Value + "\r\n")
	buf.WriteString("To: " + strings.Join(tos, ", ") + "\r\n")
	buf.WriteString("Subject: " + email.Subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + w.Boundary() + "\"\r\n\r\n")

	if email.Text != "" {
		part, err := w.CreatePart(map[string][]string{"Content-Type": {"text/plain; charset=UTF-8"}})
		if err != nil {
			return nil, err
		}
		if _, err = part.Write([]byte(email.Text)); err != nil {
			return nil, err
		}
	}
	if email.HTML != "" {
		part, err := w.CreatePart(map[string][]string{"Content-Type": {"text/html; charset=UTF-8"}})
		if err != nil {
			return nil, err
		}
		if _, err = part.Write([]byte(email.HTML)); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var _ emailport.EmailPort = (*Client)(nil)
