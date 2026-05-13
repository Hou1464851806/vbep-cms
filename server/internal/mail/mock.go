package mail

import (
	"fmt"
	"server/logger"
)

type MockSender struct{}

func (s *MockSender) Send(msg Message) error {
	fmt.Printf("\n========================================\n")
	logger.Debugf("[MOCK MAIL] Sending to: %s, Subject: %s, Body: %s", msg.To, msg.Subject, msg.Body)
	fmt.Printf("========================================\n\n")
	return nil
}
