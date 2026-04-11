package vault_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/secrets/vault"
	"github.com/marcusPrado02/go-commons/ports/secrets"
)

var _ secrets.Port = (*vault.Client)(nil)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *vault.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := vault.New(srv.URL, "test-token", vault.WithMount("secret"))
	return srv, client
}

func kvResponse(data map[string]any) []byte {
	envelope := map[string]any{
		"data": map[string]any{
			"data": data,
		},
	}
	b, _ := json.Marshal(envelope)
	return b
}

func TestVault_Get_SingleValueField(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "test-token" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(kvResponse(map[string]any{"value": "s3cr3t"}))
	})

	got, err := client.Get(context.Background(), "myapp/password")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "s3cr3t" {
		t.Errorf("got %q, want %q", got, "s3cr3t")
	}
}

func TestVault_Get_MultiFieldReturnsJSON(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(kvResponse(map[string]any{"host": "db.internal", "port": "5432"}))
	})

	got, err := client.Get(context.Background(), "myapp/db")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if parsed["host"] != "db.internal" {
		t.Errorf("host: got %q, want %q", parsed["host"], "db.internal")
	}
}

func TestVault_GetJSON(t *testing.T) {
	type dbConfig struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(kvResponse(map[string]any{"host": "pg.prod", "port": "5432"}))
	})

	var cfg dbConfig
	if err := client.GetJSON(context.Background(), "myapp/db", &cfg); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if cfg.Host != "pg.prod" {
		t.Errorf("Host: got %q, want %q", cfg.Host, "pg.prod")
	}
	if cfg.Port != "5432" {
		t.Errorf("Port: got %q, want %q", cfg.Port, "5432")
	}
}

func TestVault_Get_NotFound(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	_, err := client.Get(context.Background(), "nonexistent/key")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestVault_Get_Forbidden(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":["permission denied"]}`, http.StatusForbidden)
	})

	_, err := client.Get(context.Background(), "restricted/key")
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

func TestVault_Get_UnexpectedStatus(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})

	_, err := client.Get(context.Background(), "myapp/key")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestVault_Get_SendsTokenHeader(t *testing.T) {
	receivedToken := ""
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Vault-Token")
		w.Header().Set("Content-Type", "application/json")
		w.Write(kvResponse(map[string]any{"value": "ok"}))
	})

	if _, err := client.Get(context.Background(), "any/key"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if receivedToken != "test-token" {
		t.Errorf("X-Vault-Token: got %q, want %q", receivedToken, "test-token")
	}
}

func TestVault_Get_CorrectURLPath(t *testing.T) {
	receivedPath := ""
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(kvResponse(map[string]any{"value": "x"}))
	})

	if _, err := client.Get(context.Background(), "myapp/creds"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := "/v1/secret/data/myapp/creds"
	if receivedPath != want {
		t.Errorf("URL path: got %q, want %q", receivedPath, want)
	}
}
