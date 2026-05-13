package mail

import (
	"fmt"
	"server/config"
	"server/global"
)

// Message 邮件内容
type Message struct {
	To      string
	Subject string
	Body    string
}

// EmailSender 邮件发送接口，不同的驱动需要实现不同的发送函数
type EmailSender interface {
	Send(msg Message) error
}

func NewEmailSender() (EmailSender, error) {
	emailCfg := global.CFG.Email
	switch emailCfg.Driver {
	case config.DriverSmtp:
		smtpCfg := emailCfg.Smtp
		return &SmtpSender{
			Host:     smtpCfg.Host,
			Port:     smtpCfg.Port,
			User:     smtpCfg.User,
			Password: smtpCfg.Password,
			From:     emailCfg.From,
		}, nil
	case config.DriverApi:
		apiCfg := emailCfg.Api
		return &ApiSender{
			Url: apiCfg.Url,
			Key: apiCfg.Key,
		}, nil
	case "mock":
		return &MockSender{}, nil
	default:
		return nil, fmt.Errorf("unsupported email driver")
	}
}
