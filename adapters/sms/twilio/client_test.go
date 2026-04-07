package twilio_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcusPrado02/go-commons/adapters/sms/twilio"
)

// twilioRedirectTransport rewrites every request's host to the given target,
// allowing the Twilio SDK to hit a local httptest server instead of api.twilio.com.
type twilioRedirectTransport struct {
	target string
}

func (t *twilioRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.target, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func TestNew_EmptyCredentials_ReturnsError(t *testing.T) {
	cases := [][3]string{
		{"", "tok", "+1555"},
		{"sid", "", "+1555"},
		{"sid", "tok", ""},
	}
	for _, c := range cases {
		_, err := twilio.New(c[0], c[1], c[2])
		if err == nil {
			t.Errorf("expected error for credentials (%q, %q, %q)", c[0], c[1], c[2])
		}
	}
}

func TestNew_ValidCredentials_ReturnsClient(t *testing.T) {
	client, err := twilio.New("AC123", "auth123", "+15550001234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestSend_MockServer_ReturnsReceipt(t *testing.T) {
	sid := "SMtest1234"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"sid": sid})
	}))
	defer srv.Close()

	hc := &http.Client{Transport: &twilioRedirectTransport{target: srv.URL}}
	client, err := twilio.New("ACtest", "authtest", "+15550001234", twilio.WithHTTPClient(hc))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	receipt, err := client.Send(context.Background(), "+15550009999", "hello")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if receipt.MessageID != sid {
		t.Errorf("expected MessageID %q, got %q", sid, receipt.MessageID)
	}
}

func TestPing_MockServer_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sid": "ACtest", "status": "active"})
	}))
	defer srv.Close()

	hc := &http.Client{Transport: &twilioRedirectTransport{target: srv.URL}}
	client, err := twilio.New("ACtest", "authtest", "+15550001234", twilio.WithHTTPClient(hc))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
