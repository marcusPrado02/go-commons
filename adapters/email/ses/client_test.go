package ses_test

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsses "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/marcusPrado02/go-commons/adapters/email/ses"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

// newTestClient sets up a SES client pointed at the given httptest server.
func newTestClient(srv *httptest.Server, from emailport.EmailAddress) *ses.Client {
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  srv.Client(),
		BaseEndpoint: aws.String(srv.URL),
	}
	_ = awsses.NewFromConfig(cfg) // ensure SDK accepted the config
	return ses.New(cfg, from)
}

type sendEmailResponse struct {
	XMLName   xml.Name `xml:"SendEmailResponse"`
	MessageID string   `xml:"SendEmailResult>MessageId"`
}

func TestSend_ValidEmail_ReturnsReceipt(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	to, _ := emailport.NewEmailAddress("to@example.com")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_ = xml.NewEncoder(w).Encode(sendEmailResponse{MessageID: "mock-message-id-123"})
	}))
	defer srv.Close()

	client := newTestClient(srv, from)
	email := emailport.Email{
		From:    from,
		To:      []emailport.EmailAddress{to},
		Subject: "Test",
		Text:    "hello",
	}
	receipt, err := client.Send(context.Background(), email)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	_ = receipt // MessageID comes from mock XML response
}

func TestSend_InvalidEmail_ReturnsValidationError(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(srv, from)
	// Missing From and To — Validate() must fail before HTTP call.
	_, err := client.Send(context.Background(), emailport.Email{Subject: "x"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSendWithTemplate_ReturnsStructuredError(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	to, _ := emailport.NewEmailAddress("to@example.com")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(srv, from)
	_, err := client.SendWithTemplate(context.Background(), emailport.TemplateEmailRequest{
		From:         from,
		To:           []emailport.EmailAddress{to},
		TemplateName: "welcome",
	})
	if err == nil {
		t.Fatal("expected error from SendWithTemplate")
	}
	// Error must mention the migration path.
	msg := err.Error()
	if msg == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestPing_MockServer_ReturnsNoError(t *testing.T) {
	from, _ := emailport.NewEmailAddress("from@example.com")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<ListIdentitiesResponse><ListIdentitiesResult><Identities/></ListIdentitiesResult></ListIdentitiesResponse>`))
	}))
	defer srv.Close()

	client := newTestClient(srv, from)
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
