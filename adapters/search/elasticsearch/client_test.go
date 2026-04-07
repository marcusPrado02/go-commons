package elasticsearch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/search/elasticsearch"
)

// esHeaders sets the headers that the Elasticsearch Go client requires to
// trust the response is from a genuine Elasticsearch node.
func esHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Elastic-Product", "Elasticsearch")
}

// newTestClient creates an Elasticsearch client pointed at the given test server.
func newTestClient(t *testing.T, srv *httptest.Server) *elasticsearch.Client {
	t.Helper()
	c, err := elasticsearch.New(elasticsearch.Config{
		Addresses: []string{srv.URL},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestIndex_ValidDocument_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		esHeaders(w)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": "created", "_id": "1"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Index(context.Background(), "test-index", "1", map[string]string{"name": "doc"})
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
}

func TestSearch_ReturnsHits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		esHeaders(w)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hits": map[string]any{
				"total": map[string]any{"value": 1},
				"hits":  []map[string]any{{"_source": map[string]string{"name": "doc"}}},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	result, err := c.Search(context.Background(), "test-index", map[string]any{"match_all": map[string]any{}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected Total=1, got %d", result.Total)
	}
	if len(result.Hits) != 1 {
		t.Errorf("expected 1 hit, got %d", len(result.Hits))
	}
}

func TestDelete_404_IsIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		esHeaders(w)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": "not_found"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Delete(context.Background(), "test-index", "missing-id"); err != nil {
		t.Fatalf("Delete: expected 404 to be ignored, got: %v", err)
	}
}

func TestPing_MockServer_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		esHeaders(w)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":    "test-node",
			"version": map[string]any{"number": "8.0.0"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
