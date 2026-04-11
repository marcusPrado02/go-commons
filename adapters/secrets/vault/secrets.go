// Package vault provides a Port implementation backed by HashiCorp Vault's KV v2 API.
// Authentication is handled externally: callers are responsible for obtaining a Vault
// token and renewing it as needed. This keeps the adapter simple and auth-method-agnostic.
package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/marcusPrado02/go-commons/ports/secrets"
)

const defaultTimeout = 10 * time.Second

// Client implements secrets.Port using HashiCorp Vault's KV v2 HTTP API.
type Client struct {
	baseURL string
	token   string
	mount   string
	http    *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the default HTTP client.
// Useful for injecting a client with custom TLS config or timeouts.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.http = c }
}

// WithMount sets the KV v2 mount path (default: "secret").
// Change this if your Vault is mounted at a different path (e.g. "kv", "app-secrets").
func WithMount(mount string) Option {
	return func(cl *Client) { cl.mount = mount }
}

// New creates a Client for the Vault server at baseURL using the given token.
// baseURL should be the root address, e.g. "https://vault.example.com".
// token is a Vault token with read access to the KV secrets.
func New(baseURL, token string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		mount:   "secret",
		http:    &http.Client{Timeout: defaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Get retrieves the latest version of the KV v2 secret at key.
// key is a path relative to the mount, e.g. "myapp/db-password".
// If the secret has multiple fields, Get returns the JSON-encoded map of all fields.
// Use GetJSON to unmarshal a structured secret directly.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	data, err := c.readSecret(ctx, key)
	if err != nil {
		return "", err
	}

	// If there is exactly one field named "value", return it as a plain string.
	if v, ok := data["value"]; ok && len(data) == 1 {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}

	// Otherwise serialise the whole map as JSON.
	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("vault: marshal secret %q: %w", key, err)
	}
	return string(raw), nil
}

// GetJSON retrieves the KV v2 secret at key and unmarshals its data fields into dest.
func (c *Client) GetJSON(ctx context.Context, key string, dest any) error {
	value, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return secrets.ParseJSON(value, dest)
}

// readSecret calls the Vault KV v2 read endpoint and returns the data map.
func (c *Client) readSecret(ctx context.Context, key string) (map[string]any, error) {
	url := fmt.Sprintf("%s/v1/%s/data/%s", c.baseURL, c.mount, strings.TrimLeft(key, "/"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("vault: build request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault: get %q: %w", key, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("vault: read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("vault: secret %q not found", key)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("vault: permission denied for %q (check token policies)", key)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault: unexpected status %d for %q: %s", resp.StatusCode, key, body)
	}

	var envelope struct {
		Data struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("vault: parse response for %q: %w", key, err)
	}
	return envelope.Data.Data, nil
}

var _ secrets.Port = (*Client)(nil)
