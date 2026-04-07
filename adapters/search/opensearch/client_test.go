package opensearch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/search/opensearch"
)

func jsonHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func newTestClient(t *testing.T, srv *httptest.Server) *opensearch.Client {
	t.Helper()
	c, err := opensearch.New(opensearch.Config{
		Addresses: []string{srv.URL},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestIndex_ValidDocument_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonHeaders(w)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": "created", "_id": "1"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Index(context.Background(), "test-index", "1", map[string]string{"name": "doc"}); err != nil {
		t.Fatalf("Index: %v", err)
	}
}

func TestSearch_ReturnsHits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonHeaders(w)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hits": map[string]any{
				"total": map[string]any{"value": 2},
				"hits": []map[string]any{
					{"_source": map[string]string{"title": "a"}},
					{"_source": map[string]string{"title": "b"}},
				},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	result, err := c.Search(context.Background(), "idx", map[string]any{"match_all": map[string]any{}})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected Total=2, got %d", result.Total)
	}
	if len(result.Hits) != 2 {
		t.Errorf("expected 2 hits, got %d", len(result.Hits))
	}
}

func TestDelete_404_IsIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonHeaders(w)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"result": "not_found"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Delete(context.Background(), "idx", "no-such-id"); err != nil {
		t.Fatalf("Delete 404: %v", err)
	}
}

func TestPing_MockServer_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonHeaders(w)
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "node", "version": map[string]any{"number": "2.0.0"}})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
