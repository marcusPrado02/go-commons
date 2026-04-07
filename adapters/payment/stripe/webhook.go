package stripe

import (
	"fmt"

	stripeapi "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
)

// VerifyWebhookSignature validates a Stripe webhook payload against the given
// signature header and endpoint secret. Returns the parsed event on success.
//
// payload is the raw request body bytes (do not JSON-parse before passing).
// sigHeader is the value of the "Stripe-Signature" HTTP header.
// secret is the webhook endpoint secret from the Stripe dashboard.
func VerifyWebhookSignature(payload []byte, sigHeader, secret string) (stripeapi.Event, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, secret)
	if err != nil {
		return stripeapi.Event{}, fmt.Errorf("stripe: webhook signature verification failed: %w", err)
	}
	return event, nil
}
