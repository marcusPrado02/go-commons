package contracts

import (
	"context"

	"github.com/marcusPrado02/go-commons/ports/email"
	"github.com/stretchr/testify/suite"
)

// EmailContract is a reusable test suite for Port implementations.
// Embed it in your adapter test and provide a working Port and valid addresses.
//
// Example:
//
//	func TestSMTPClient(t *testing.T) {
//	    suite.Run(t, &contracts.EmailContract{
//	        Port: mysmtp.NewClient(...),
//	        From: emailport.Address{Value: "from@example.com"},
//	        To:   emailport.Address{Value: "to@example.com"},
//	    })
//	}
type EmailContract struct {
	suite.Suite
	// Port is the Port implementation under test.
	Port email.Port
	// From is the sender address used in test messages.
	From email.Address
	// To is the recipient address used in test messages.
	To email.Address
}

func (s *EmailContract) TestSend_ValidEmail_ReturnsReceipt() {
	msg := email.Email{
		From:    s.From,
		To:      []email.Address{s.To},
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
