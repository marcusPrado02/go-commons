package sesv2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	awssesv2 "github.com/aws/aws-sdk-go-v2/service/sesv2"

	"github.com/marcusPrado02/go-commons/adapters/email/sesv2"
	emailport "github.com/marcusPrado02/go-commons/ports/email"
)

var _ emailport.Port = (*sesv2.Client)(nil)

func newTestClient(t *testing.T, handler http.HandlerFunc) *sesv2.Client {
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

	from, _ := emailport.NewEmailAddress("sender@example.com")
	return sesv2.NewWithOptions(cfg, from, func(o *awssesv2.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
	})
}

func sendMessageResponse(messageID string) []byte {
	b, _ := json.Marshal(map[string]any{"MessageId": messageID})
	return b
}

func TestSESv2_Send(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "outbound-emails") {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(sendMessageResponse("msg-sesv2-001"))
	})

	from, _ := emailport.NewEmailAddress("sender@example.com")
	to, _ := emailport.NewEmailAddress("recipient@example.com")

	receipt, err := client.Send(context.Background(), emailport.Email{
		From:    from,
		To:      []emailport.Address{to},
		Subject: "Hello",
		Text:    "World",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if receipt.MessageID != "msg-sesv2-001" {
		t.Errorf("MessageID: got %q, want %q", receipt.MessageID, "msg-sesv2-001")
	}
}

func TestSESv2_Send_Validation(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	from, _ := emailport.NewEmailAddress("sender@example.com")

	// No recipients — must fail before hitting the network.
	_, err := client.Send(context.Background(), emailport.Email{
		From:    from,
		To:      nil,
		Subject: "Bad",
		Text:    "no recipients",
	})
	if err == nil {
		t.Fatal("expected validation error for email with no recipients, got nil")
	}
}

func TestSESv2_Send_NoBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	from, _ := emailport.NewEmailAddress("sender@example.com")
	to, _ := emailport.NewEmailAddress("recipient@example.com")

	_, err := client.Send(context.Background(), emailport.Email{
		From:    from,
		To:      []emailport.Address{to},
		Subject: "Empty body",
		// HTML and Text both empty — should fail validation.
	})
	if err == nil {
		t.Fatal("expected validation error for email with no body, got nil")
	}
}

func TestSESv2_SendWithTemplate(t *testing.T) {
	var capturedBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write(sendMessageResponse("msg-template-001"))
	})

	from, _ := emailport.NewEmailAddress("sender@example.com")
	to, _ := emailport.NewEmailAddress("user@example.com")

	receipt, err := client.SendWithTemplate(context.Background(), emailport.TemplateEmailRequest{
		From:         from,
		To:           []emailport.Address{to},
		TemplateName: "welcome-email",
		Variables:    map[string]any{"username": "Alice", "plan": "pro"},
	})
	if err != nil {
		t.Fatalf("SendWithTemplate: %v", err)
	}
	if receipt.MessageID != "msg-template-001" {
		t.Errorf("MessageID: got %q, want %q", receipt.MessageID, "msg-template-001")
	}
}

func TestSESv2_SendWithTemplate_NoRecipients(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	from, _ := emailport.NewEmailAddress("sender@example.com")
	_, err := client.SendWithTemplate(context.Background(), emailport.TemplateEmailRequest{
		From:         from,
		To:           nil,
		TemplateName: "welcome-email",
	})
	if err == nil {
		t.Fatal("expected error for SendWithTemplate with no recipients, got nil")
	}
}

func TestSESv2_Ping(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"ContactLists":[]}`)
	})

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
