package service

import (
	"fmt"
	"github.com/hibiken/asynq"
	"server/internal/mail"
	"server/internal/tasks"
)

// MailService 向asynq队列发送邮件
type MailService struct {
	client *asynq.Client
}

func NewMailService(client *asynq.Client) *MailService {
	return &MailService{
		client: client,
	}
}

func (s *MailService) SendResetPassword(target, subject, content string) error {
	msg := mail.Message{
		To:      target,
		Subject: subject,
		Body:    content,
	}
	task, err := tasks.NewEmailDeliveryTask(msg)
	if err != nil {
		return err
	}
	_, err = s.client.Enqueue(task, asynq.Queue("critical"), asynq.MaxRetry(3))
	return err
}

func (s *MailService) SendNotification(target string, subject string, content string) error {
	body := fmt.Sprintf("收到一条新的系统提醒：<br/> %s", content)
	msg := mail.Message{
		To:      target,
		Subject: subject,
		Body:    body,
	}
	task, err := tasks.NewEmailDeliveryTask(msg)
	if err != nil {
		return err
	}
	_, err = s.client.Enqueue(task, asynq.Queue("default"))
	return err
}
