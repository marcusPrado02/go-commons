// Package elasticsearch provides an Elasticsearch implementation for search operations.
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

// Client wraps the Elasticsearch client for indexing and searching.
type Client struct {
	es *elasticsearch.Client
}

// Config holds Elasticsearch connection configuration.
type Config struct {
	Addresses []string
	Username  string
	Password  string
}

// New creates a new Elasticsearch client.
func New(cfg Config) (*Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: failed to create client: %w", err)
	}
	return &Client{es: es}, nil
}

// Index stores a document in the given index with the given ID.
func (c *Client) Index(_ context.Context, index, id string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("elasticsearch: marshal failed: %w", err)
	}
	res, err := c.es.Index(index, bytes.NewReader(body),
		c.es.Index.WithDocumentID(id),
		c.es.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch: index failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch: index error: %s", res.Status())
	}
	return nil
}

// SearchResult holds raw search results from Elasticsearch.
type SearchResult struct {
	Total int
	Hits  []json.RawMessage
}

// Search executes a query against the given index.
func (c *Client) Search(_ context.Context, index string, query map[string]any) (SearchResult, error) {
	body, err := json.Marshal(map[string]any{"query": query})
	if err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: marshal query failed: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithIndex(index),
		c.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return SearchResult{}, fmt.Errorf("elasticsearch: search error: %s", res.Status())
	}

	var result struct {
		Hits struct {
			Total struct{ Value int }
			Hits  []struct {
				Source json.RawMessage `json:"_source"`
			}
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return SearchResult{}, fmt.Errorf("elasticsearch: decode failed: %w", err)
	}

	hits := make([]json.RawMessage, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		hits[i] = h.Source
	}
	return SearchResult{Total: result.Hits.Total.Value, Hits: hits}, nil
}

// Delete removes a document by ID from the given index.
func (c *Client) Delete(_ context.Context, index, id string) error {
	res, err := c.es.Delete(index, id)
	if err != nil {
		return fmt.Errorf("elasticsearch: delete failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() && !strings.Contains(res.Status(), "404") {
		return fmt.Errorf("elasticsearch: delete error: %s", res.Status())
	}
	return nil
}

// Ping verifies Elasticsearch connectivity.
func (c *Client) Ping(_ context.Context) error {
	res, err := c.es.Ping()
	if err != nil {
		return fmt.Errorf("elasticsearch: ping failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch: ping error: %s", res.Status())
	}
	return nil
}
