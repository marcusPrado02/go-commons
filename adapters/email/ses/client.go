// Package ses provides an AWS SES implementation of ports/email.Port.
package ses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsses "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	kerrors "github.com/marcusPrado02/go-commons/kernel/errors"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

// Client is an AWS SES implementation of Port.
type Client struct {
	ses  *awsses.Client
	from emailport.Address
}

// New creates a new SES client.
func New(cfg aws.Config, from emailport.Address) *Client {
	return &Client{ses: awsses.NewFromConfig(cfg), from: from}
}

// Send delivers an email via AWS SES.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.Receipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.Receipt{}, fmt.Errorf("ses: %w", err)
	}

	tos := make([]string, len(email.To))
	for i, t := range email.To {
		tos[i] = t.Value
	}

	body := &types.Body{}
	if email.HTML != "" {
		body.Html = &types.Content{Data: aws.String(email.HTML), Charset: aws.String("UTF-8")}
	}
	if email.Text != "" {
		body.Text = &types.Content{Data: aws.String(email.Text), Charset: aws.String("UTF-8")}
	}

	out, err := c.ses.SendEmail(ctx, &awsses.SendEmailInput{
		Source:      aws.String(email.From.Value),
		Destination: &types.Destination{ToAddresses: tos},
		Message: &types.Message{
			Subject: &types.Content{Data: aws.String(email.Subject), Charset: aws.String("UTF-8")},
			Body:    body,
		},
	})
	if err != nil {
		return emailport.Receipt{}, fmt.Errorf("ses: send failed: %w", err)
	}
	return emailport.Receipt{MessageID: aws.ToString(out.MessageId)}, nil
}

// SendWithTemplate is not supported by SES v1 — use adapters/email/sesv2 instead.
func (c *Client) SendWithTemplate(_ context.Context, req emailport.TemplateEmailRequest) (emailport.Receipt, error) {
	return emailport.Receipt{}, kerrors.ErrTechnical.
		WithDetail("reason", "SES v1 does not support template sending").
		WithDetail("template", req.TemplateName).
		WithDetail("migrate_to", "adapters/email/sesv2").
		WithCause(fmt.Errorf("ses: SendWithTemplate not supported — migrate to adapters/email/sesv2"))
}

// Ping verifies SES connectivity by listing identities.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.ses.ListIdentities(ctx, &awsses.ListIdentitiesInput{MaxItems: aws.Int32(1)})
	if err != nil {
		return fmt.Errorf("ses: ping failed: %w", err)
	}
	return nil
}

var _ emailport.Port = (*Client)(nil)
