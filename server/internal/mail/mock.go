package mail

import (
	"fmt"
	"server/logger"
)

type MockSender struct{}

func (s *MockSender) Send(msg Message) error {
	fmt.Printf("\n========================================\n")
	logger.Debugw("[MOCK MAIL] Sending to, Subject, Body", "to", msg.To, "subject", msg.Subject, "body", msg.Body)
	fmt.Printf("========================================\n\n")
	return nil
}
