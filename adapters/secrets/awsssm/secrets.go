// Package awsssm provides a Port implementation backed by AWS SSM Parameter Store.
package awsssm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/marcusPrado02/go-commons/ports/secrets"
)

// Client implements secrets.Port using AWS SSM Parameter Store.
// Parameters are fetched with decryption enabled (WithDecryption: true), so both
// String and SecureString parameter types are supported transparently.
type Client struct {
	ssm *ssm.Client
}

// New creates a Client from an existing aws.Config.
func New(cfg aws.Config) *Client {
	return &Client{ssm: ssm.NewFromConfig(cfg)}
}

// Get retrieves the decrypted value of the SSM parameter at the given path/key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	out, err := c.ssm.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(key),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("awsssm: get %q: %w", key, err)
	}
	if out.Parameter == nil || out.Parameter.Value == nil {
		return "", fmt.Errorf("awsssm: parameter %q returned nil value", key)
	}
	return aws.ToString(out.Parameter.Value), nil
}

// GetJSON retrieves the SSM parameter and unmarshals its JSON value into dest.
func (c *Client) GetJSON(ctx context.Context, key string, dest any) error {
	value, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return secrets.ParseJSON(value, dest)
}

var _ secrets.Port = (*Client)(nil)
