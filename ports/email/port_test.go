package email_test

import (
	"testing"

	"github.com/marcusPrado02/go-commons/ports/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailAddress_valid(t *testing.T) {
	addr, err := email.NewEmailAddress("user@example.com")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", addr.Value)
}

func TestNewEmailAddress_withDisplayName(t *testing.T) {
	addr, err := email.NewEmailAddress("Alice <alice@example.com>")
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", addr.Value)
}

func TestNewEmailAddress_invalid(t *testing.T) {
	_, err := email.NewEmailAddress("not-an-email")
	assert.Error(t, err)
}

func TestEmail_Validate_valid(t *testing.T) {
	from, _ := email.NewEmailAddress("sender@example.com")
	to, _ := email.NewEmailAddress("recipient@example.com")
	e := email.Email{From: from, To: []email.EmailAddress{to}, HTML: "<p>hello</p>"}
	assert.NoError(t, e.Validate())
}

func TestEmail_Validate_missingTo(t *testing.T) {
	from, _ := email.NewEmailAddress("sender@example.com")
	e := email.Email{From: from, HTML: "<p>hello</p>"}
	assert.Error(t, e.Validate())
}

func TestEmail_Validate_missingBody(t *testing.T) {
	from, _ := email.NewEmailAddress("sender@example.com")
	to, _ := email.NewEmailAddress("r@example.com")
	e := email.Email{From: from, To: []email.EmailAddress{to}}
	assert.Error(t, e.Validate())
}

func TestEmail_Validate_missingFrom(t *testing.T) {
	to, _ := email.NewEmailAddress("r@example.com")
	e := email.Email{To: []email.EmailAddress{to}, Text: "hello"}
	assert.Error(t, e.Validate())
}
