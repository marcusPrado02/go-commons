package awsssm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"

	ssmadapter "github.com/marcusPrado02/go-commons/adapters/secrets/awsssm"
	"github.com/marcusPrado02/go-commons/ports/secrets"
)

var _ secrets.Port = (*ssmadapter.Client)(nil)

// ssmJSONResponse builds an SSM GetParameter JSON response body.
func ssmJSONResponse(name, value string) []byte {
	b, _ := json.Marshal(map[string]any{
		"Parameter": map[string]any{
			"Name":  name,
			"Value": value,
			"Type":  "SecureString",
		},
	})
	return b
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *ssmadapter.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.BaseEndpoint = aws.String(srv.URL)
	return ssmadapter.New(cfg)
}

func TestSSM_Get(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(ssmJSONResponse("/myapp/db-password", "hunter2"))
	})

	got, err := client.Get(context.Background(), "/myapp/db-password")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "hunter2" {
		t.Errorf("got %q, want %q", got, "hunter2")
	}
}

func TestSSM_GetJSON(t *testing.T) {
	type cfg struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	payload, _ := json.Marshal(cfg{Host: "db.prod", Port: 5432})
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(ssmJSONResponse("/myapp/db", string(payload)))
	})

	var got cfg
	if err := client.GetJSON(context.Background(), "/myapp/db", &got); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if got.Host != "db.prod" {
		t.Errorf("Host: got %q, want %q", got.Host, "db.prod")
	}
	if got.Port != 5432 {
		t.Errorf("Port: got %d, want %d", got.Port, 5432)
	}
}

func TestSSM_Get_ErrorResponse(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, `{"__type":"ParameterNotFound","message":"Parameter not found"}`)
	})

	_, err := client.Get(context.Background(), "/nonexistent")
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}
}

func TestSSM_Get_SendsWithDecryption(t *testing.T) {
	withDecryption := false
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name           string `json:"Name"`
			WithDecryption bool   `json:"WithDecryption"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		withDecryption = body.WithDecryption
		w.Header().Set("Content-Type", "application/json")
		w.Write(ssmJSONResponse(body.Name, "decrypted-value"))
	})

	if _, err := client.Get(context.Background(), "/myapp/secret"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !withDecryption {
		t.Error("expected WithDecryption=true in the SSM request")
	}
}
