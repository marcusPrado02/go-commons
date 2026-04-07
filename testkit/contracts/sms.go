package contracts

import (
	"context"

	"github.com/marcusPrado02/go-commons/ports/sms"
	"github.com/stretchr/testify/suite"
)

// SMSContract is a reusable test suite for SMSPort implementations.
//
// Example:
//
//	func TestTwilioClient(t *testing.T) {
//	    suite.Run(t, &contracts.SMSContract{
//	        Port: twilio.New(...),
//	        To:   "+15550001234",
//	    })
//	}
type SMSContract struct {
	suite.Suite
	// Port is the SMSPort implementation under test.
	Port sms.SMSPort
	// To is a valid E.164 phone number used in test sends.
	To string
}

func (s *SMSContract) TestSend_ValidNumber_ReturnsReceipt() {
	receipt, err := s.Port.Send(context.Background(), s.To, "contract test message")
	s.Require().NoError(err)
	_ = receipt
}

func (s *SMSContract) TestSend_EmptyTo_ReturnsError() {
	_, err := s.Port.Send(context.Background(), "", "body")
	s.Require().Error(err, "expected error for empty recipient")
}

func (s *SMSContract) TestPing_ReturnsNoError() {
	s.Require().NoError(s.Port.Ping(context.Background()))
}
