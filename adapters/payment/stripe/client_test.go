package stripe_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	stripeapi "github.com/stripe/stripe-go/v76"
	"github.com/marcusPrado02/go-commons/adapters/payment/stripe"
)

func TestNew_EmptyAPIKey_ReturnsError(t *testing.T) {
	_, err := stripe.New("")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestNew_ValidAPIKey_ReturnsClient(t *testing.T) {
	c, err := stripe.New("sk_test_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestCreatePaymentIntent_ZeroAmount_ReturnsError(t *testing.T) {
	c, _ := stripe.New("sk_test_abc")
	_, err := c.CreatePaymentIntent(context.Background(), 0, "usd", "test")
	if err == nil {
		t.Fatal("expected error for amount=0")
	}
}

func TestCreatePaymentIntent_NegativeAmount_ReturnsError(t *testing.T) {
	c, _ := stripe.New("sk_test_abc")
	_, err := c.CreatePaymentIntent(context.Background(), -100, "usd", "test")
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestRefund_EmptyChargeID_ReturnsError(t *testing.T) {
	c, _ := stripe.New("sk_test_abc")
	_, err := c.Refund(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty chargeID")
	}
}

func TestCreatePaymentIntent_MockServer_ReturnsReceipt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            "pi_mock123",
			"client_secret": "pi_mock123_secret_abc",
			"status":        "requires_payment_method",
			"object":        "payment_intent",
		})
	}))
	defer srv.Close()

	// Redirect all Stripe API requests to our test server via a custom backend.
	u := srv.URL
	backend := stripeapi.GetBackendWithConfig(stripeapi.APIBackend, &stripeapi.BackendConfig{
		URL:        &u,
		HTTPClient: srv.Client(),
	})
	stripeapi.SetBackend(stripeapi.APIBackend, backend)

	c, _ := stripe.New("sk_test_mock")
	result, err := c.CreatePaymentIntent(context.Background(), 1000, "usd", "mock payment")
	if err != nil {
		t.Fatalf("CreatePaymentIntent: %v", err)
	}
	if result.ID != "pi_mock123" {
		t.Errorf("expected ID %q, got %q", "pi_mock123", result.ID)
	}
}
