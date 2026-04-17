package mail

import (
	"fmt"
	"github.com/wneessen/go-mail"
	"strings"
	"time"
)

type SmtpSender struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func (s *SmtpSender) Send(msg Message) error {
	if strings.TrimSpace(s.Host) == "" || s.Port <= 0 {
		return fmt.Errorf("invalid smtp config: host=%q port=%d", s.Host, s.Port)
	}
	if strings.TrimSpace(s.From) == "" {
		return fmt.Errorf("invalid smtp config: from address is empty")
	}
	if strings.TrimSpace(msg.To) == "" {
		return fmt.Errorf("invalid mail message: to address is empty")
	}

	m := mail.NewMsg()
	if err := m.To(msg.To); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}
	if err := m.From(s.From); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	m.Subject(msg.Subject)
	m.SetBodyString(mail.TypeTextHTML, msg.Body)

	opts := []mail.Option{
		mail.WithPort(s.Port),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithUsername(s.User),
		mail.WithPassword(s.Password),
		mail.WithTimeout(10 * time.Second),
	}

	switch s.Port {
	case 465:
		opts = append(opts, mail.WithSSL())
	case 587:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	default:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	}

	c, err := mail.NewClient(s.Host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create new email delivery client: %s", err)
	}
	if err := c.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %s", err)
	}
	return nil
}
