// Package stripe provides a Stripe payment adapter.
// Implements common payment operations: create intent, confirm, and refund.
package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
)

// Client wraps the Stripe SDK for payment operations.
type Client struct {
	apiKey string
}

// New creates a new Stripe client.
func New(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("stripe: apiKey cannot be empty")
	}
	stripe.Key = apiKey
	return &Client{apiKey: apiKey}, nil
}

// PaymentIntentResult holds the result of a created payment intent.
type PaymentIntentResult struct {
	ID           string
	ClientSecret string
	Status       string
}

// CreatePaymentIntent creates a new Stripe PaymentIntent.
// Amount must be > 0 (smallest currency unit, e.g. cents for USD).
func (c *Client) CreatePaymentIntent(_ context.Context, amount int64, currency, description string) (PaymentIntentResult, error) {
	if amount <= 0 {
		return PaymentIntentResult{}, fmt.Errorf("stripe: amount must be > 0, got %d", amount)
	}
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(amount),
		Currency:    stripe.String(currency),
		Description: stripe.String(description),
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		return PaymentIntentResult{}, fmt.Errorf("stripe: create payment intent failed: %w", err)
	}
	return PaymentIntentResult{
		ID:           pi.ID,
		ClientSecret: pi.ClientSecret,
		Status:       string(pi.Status),
	}, nil
}

// ConfirmPaymentIntent confirms a payment intent with the given payment method.
func (c *Client) ConfirmPaymentIntent(_ context.Context, intentID, paymentMethodID string) (PaymentIntentResult, error) {
	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(paymentMethodID),
	}
	pi, err := paymentintent.Confirm(intentID, params)
	if err != nil {
		return PaymentIntentResult{}, fmt.Errorf("stripe: confirm payment intent failed: %w", err)
	}
	return PaymentIntentResult{ID: pi.ID, Status: string(pi.Status)}, nil
}

// RefundResult holds the result of a refund operation.
type RefundResult struct {
	ID     string
	Status string
}

// Refund creates a full refund for the given charge.
func (c *Client) Refund(_ context.Context, chargeID string) (RefundResult, error) {
	if chargeID == "" {
		return RefundResult{}, fmt.Errorf("stripe: chargeID cannot be empty")
	}
	params := &stripe.RefundParams{Charge: stripe.String(chargeID)}
	r, err := refund.New(params)
	if err != nil {
		return RefundResult{}, fmt.Errorf("stripe: refund failed: %w", err)
	}
	return RefundResult{ID: r.ID, Status: string(r.Status)}, nil
}
