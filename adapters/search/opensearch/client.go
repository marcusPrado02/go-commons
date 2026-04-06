// Package opensearch provides an OpenSearch implementation for search operations.
// The API mirrors the elasticsearch adapter — same operations, different SDK.
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
)

// Client wraps the OpenSearch client.
type Client struct {
	os *opensearch.Client
}

// Config holds OpenSearch connection configuration.
type Config struct {
	Addresses []string
	Username  string
	Password  string
}

// New creates a new OpenSearch client.
func New(cfg Config) (*Client, error) {
	os, err := opensearch.NewClient(opensearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("opensearch: failed to create client: %w", err)
	}
	return &Client{os: os}, nil
}

// Index stores a document.
func (c *Client) Index(_ context.Context, index, id string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("opensearch: marshal failed: %w", err)
	}
	res, err := c.os.Index(index, bytes.NewReader(body),
		c.os.Index.WithDocumentID(id),
		c.os.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("opensearch: index failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("opensearch: index error: %s", res.Status())
	}
	return nil
}

// SearchResult holds raw search hits.
type SearchResult struct {
	Total int
	Hits  []json.RawMessage
}

// Search executes a query.
func (c *Client) Search(_ context.Context, index string, query map[string]any) (SearchResult, error) {
	body, _ := json.Marshal(map[string]any{"query": query})
	res, err := c.os.Search(
		c.os.Search.WithIndex(index),
		c.os.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return SearchResult{}, fmt.Errorf("opensearch: search failed: %w", err)
	}
	defer res.Body.Close()

	var result struct {
		Hits struct {
			Total struct{ Value int }
			Hits  []struct {
				Source json.RawMessage `json:"_source"`
			}
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return SearchResult{}, err
	}
	hits := make([]json.RawMessage, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		hits[i] = h.Source
	}
	return SearchResult{Total: result.Hits.Total.Value, Hits: hits}, nil
}

// Delete removes a document by ID.
func (c *Client) Delete(_ context.Context, index, id string) error {
	res, err := c.os.Delete(index, id)
	if err != nil {
		return fmt.Errorf("opensearch: delete failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() && !strings.Contains(res.Status(), "404") {
		return fmt.Errorf("opensearch: delete error: %s", res.Status())
	}
	return nil
}

// Ping verifies OpenSearch connectivity.
func (c *Client) Ping(_ context.Context) error {
	res, err := c.os.Ping()
	if err != nil {
		return fmt.Errorf("opensearch: ping failed: %w", err)
	}
	defer res.Body.Close()
	return nil
}
