package contracts

import (
	"context"

	"github.com/marcusPrado02/go-commons/ports/email"
	"github.com/stretchr/testify/suite"
)

// EmailContract is a reusable test suite for EmailPort implementations.
// Embed it in your adapter test and provide a working Port and valid addresses.
//
// Example:
//
//	func TestSMTPClient(t *testing.T) {
//	    suite.Run(t, &contracts.EmailContract{
//	        Port: mysmtp.NewClient(...),
//	        From: emailport.EmailAddress{Value: "from@example.com"},
//	        To:   emailport.EmailAddress{Value: "to@example.com"},
//	    })
//	}
type EmailContract struct {
	suite.Suite
	// Port is the EmailPort implementation under test.
	Port email.EmailPort
	// From is the sender address used in test messages.
	From email.EmailAddress
	// To is the recipient address used in test messages.
	To email.EmailAddress
}

func (s *EmailContract) TestSend_ValidEmail_ReturnsReceipt() {
	msg := email.Email{
		From:    s.From,
		To:      []email.EmailAddress{s.To},
		Subject: "Contract test",
		Text:    "Hello from the contract suite.",
	}
	_, err := s.Port.Send(context.Background(), msg)
	s.Require().NoError(err)
}

func (s *EmailContract) TestSend_InvalidEmail_ReturnsError() {
	// Missing From and To — Validate() must reject this.
	msg := email.Email{Subject: "Invalid"}
	_, err := s.Port.Send(context.Background(), msg)
	s.Require().Error(err, "expected error for invalid email")
}

func (s *EmailContract) TestPing_ReturnsNoError() {
	s.Require().NoError(s.Port.Ping(context.Background()))
}
