// Package sesv2 provides an AWS SES v2 implementation of ports/email.EmailPort.
// Unlike adapters/email/ses (SES v1), this adapter fully supports SendWithTemplate.
package sesv2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssesv2 "github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

// Client is an AWS SES v2 implementation of EmailPort.
type Client struct {
	ses  *awssesv2.Client
	from emailport.EmailAddress
}

// New creates a Client from an existing aws.Config and a default From address.
func New(cfg aws.Config, from emailport.EmailAddress) *Client {
	return &Client{ses: awssesv2.NewFromConfig(cfg), from: from}
}

// NewWithOptions creates a Client with additional SES v2 options (e.g. a custom endpoint for tests).
func NewWithOptions(cfg aws.Config, from emailport.EmailAddress, opts ...func(*awssesv2.Options)) *Client {
	return &Client{ses: awssesv2.NewFromConfig(cfg, opts...), from: from}
}

// Send delivers an email via AWS SES v2.
func (c *Client) Send(ctx context.Context, email emailport.Email) (emailport.EmailReceipt, error) {
	if err := email.Validate(); err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sesv2: %w", err)
	}

	dest := buildDestination(email)
	content := &types.EmailContent{
		Simple: &types.Message{
			Subject: &types.Content{Data: aws.String(email.Subject), Charset: aws.String("UTF-8")},
			Body:    buildBody(email),
		},
	}

	out, err := c.ses.SendEmail(ctx, &awssesv2.SendEmailInput{
		FromEmailAddress: aws.String(email.From.Value),
		Destination:      dest,
		Content:          content,
	})
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sesv2: send: %w", err)
	}
	return emailport.EmailReceipt{MessageID: aws.ToString(out.MessageId)}, nil
}

// SendWithTemplate delivers an email using a SES v2 template.
// Variables in req.Variables are serialised as a JSON object and passed as
// TemplateData (SES v2 uses Handlebars-style {{variable}} substitution).
func (c *Client) SendWithTemplate(ctx context.Context, req emailport.TemplateEmailRequest) (emailport.EmailReceipt, error) {
	if len(req.To) == 0 {
		return emailport.EmailReceipt{}, fmt.Errorf("sesv2: SendWithTemplate requires at least one recipient")
	}

	templateData, err := json.Marshal(req.Variables)
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sesv2: marshal template variables: %w", err)
	}

	tos := make([]string, len(req.To))
	for i, t := range req.To {
		tos[i] = t.Value
	}

	from := req.From.Value
	if from == "" {
		from = c.from.Value
	}

	out, err := c.ses.SendEmail(ctx, &awssesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination:      &types.Destination{ToAddresses: tos},
		Content: &types.EmailContent{
			Template: &types.Template{
				TemplateName: aws.String(req.TemplateName),
				TemplateData: aws.String(string(templateData)),
			},
		},
	})
	if err != nil {
		return emailport.EmailReceipt{}, fmt.Errorf("sesv2: send with template %q: %w", req.TemplateName, err)
	}
	return emailport.EmailReceipt{MessageID: aws.ToString(out.MessageId)}, nil
}

// Ping verifies SES v2 connectivity by listing contact lists (lightweight read-only call).
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.ses.ListContactLists(ctx, &awssesv2.ListContactListsInput{PageSize: aws.Int32(1)})
	if err != nil {
		return fmt.Errorf("sesv2: ping: %w", err)
	}
	return nil
}

func buildDestination(email emailport.Email) *types.Destination {
	dest := &types.Destination{
		ToAddresses: make([]string, len(email.To)),
	}
	for i, t := range email.To {
		dest.ToAddresses[i] = t.Value
	}
	for _, cc := range email.CC {
		dest.CcAddresses = append(dest.CcAddresses, cc.Value)
	}
	for _, bcc := range email.BCC {
		dest.BccAddresses = append(dest.BccAddresses, bcc.Value)
	}
	return dest
}

func buildBody(email emailport.Email) *types.Body {
	body := &types.Body{}
	if email.HTML != "" {
		body.Html = &types.Content{Data: aws.String(email.HTML), Charset: aws.String("UTF-8")}
	}
	if email.Text != "" {
		body.Text = &types.Content{Data: aws.String(email.Text), Charset: aws.String("UTF-8")}
	}
	return body
}

var _ emailport.EmailPort = (*Client)(nil)
