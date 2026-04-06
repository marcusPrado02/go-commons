// Package ses provides an AWS SES implementation of ports/email.EmailPort.
package ses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsses "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

// Client is an AWS SES implementation of EmailPort.
type Client struct {
	ses  *awsses.Client
	from emailport.EmailAddress
}

// New creates a new SES client.
func New(cfg aws.Config, from emailport.EmailAddress) *Client {
	return &Client{ses: awsses.NewFromConfig(cfg), from: from}
}

// Send delivers an email via AWS SES.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("ses: %w", err)
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
		return emailport.EmailReceipt{}, fmt.Errorf("ses: send failed: %w", err)
	}
	return emailport.EmailReceipt{MessageID: aws.ToString(out.MessageId)}, nil
}

// SendWithTemplate is not supported by SES v1 — returns unsupported error.
func (c *Client) SendWithTemplate(_ context.Context, _ emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	return emailport.EmailReceipt{}, fmt.Errorf("ses: SendWithTemplate requires SES v2 — use the sesv2 adapter")
}

// Ping verifies SES connectivity by listing identities.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.ses.ListIdentities(ctx, &awsses.ListIdentitiesInput{MaxItems: aws.Int32(1)})
	if err != nil {
		return fmt.Errorf("ses: ping failed: %w", err)
	}
	return nil
}

var _ emailport.EmailPort = (*Client)(nil)
