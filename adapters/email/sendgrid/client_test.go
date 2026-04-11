package sendgrid_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/email/sendgrid"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Message-Id", "test-msg-id")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

func TestClient_Send_Success(t *testing.T) {
	srv := newTestServer(http.StatusAccepted, "")
	defer srv.Close()

	from, _ := emailport.NewEmailAddress("sender@example.com")
	client, err := sendgrid.New("test-api-key", from, sendgrid.WithBaseURL(srv.URL))
	require.NoError(t, err)

	to, _ := emailport.NewEmailAddress("recipient@example.com")
	receipt, err := client.Send(context.Background(), emailport.Email{
		From:    from,
		To:      []emailport.Address{to},
		Subject: "Hello",
		HTML:    "<p>Hi</p>",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, receipt.MessageID)
}

func TestClient_Send_ValidationFailure(t *testing.T) {
	from, _ := emailport.NewEmailAddress("sender@example.com")
	client, _ := sendgrid.New("test-api-key", from)

	// Email with no recipients — Validate() should fail before HTTP call
	_, err := client.Send(context.Background(), emailport.Email{
		From:    from,
		Subject: "Bad email",
		HTML:    "<p>Hi</p>",
	})
	assert.Error(t, err)
}

func TestClient_Ping_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"enabled": true}})
	}))
	defer srv.Close()

	from, _ := emailport.NewEmailAddress("sender@example.com")
	client, _ := sendgrid.New("test-api-key", from, sendgrid.WithBaseURL(srv.URL))

	err := client.Ping(context.Background())
	assert.NoError(t, err)
}

// Compile-time interface check
var _ emailport.Port = (*sendgrid.Client)(nil)
