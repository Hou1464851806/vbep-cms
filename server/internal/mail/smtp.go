package mail

import (
	"fmt"
	"github.com/wneessen/go-mail"
)

type SmtpSender struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func (s *SmtpSender) Send(msg Message) error {
	m := mail.NewMsg()
	if err := m.To(msg.To); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}
	if err := m.From(s.From); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	m.Subject(msg.Subject)
	m.SetBodyString(mail.TypeTextHTML, msg.Body)

	c, err := mail.NewClient(s.Host,
		mail.WithPort(s.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(s.User),
		mail.WithPassword(s.Password),
	)
	if err != nil {
		return fmt.Errorf("failed to create new email delivery client: %s", err)
	}
	if err := c.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %s", err)
	}
	return nil
}
