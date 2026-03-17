package service

import (
	"fmt"
	"github.com/hibiken/asynq"
	"server/internal/mail"
	"server/internal/tasks"
	"server/internal/templates"
)

// MailService 向asynq队列发送邮件
type MailService struct {
	client *asynq.Client
	engine *templates.TemplateEngine
}

func NewMailService(client *asynq.Client, engine *templates.TemplateEngine) *MailService {
	return &MailService{
		client: client,
		engine: engine,
	}
}

func (s *MailService) pushNewMail(msg mail.Message, queueName string, maxRetry int) error {
	task, err := tasks.NewEmailDeliveryTask(msg)
	if err != nil {
		return err
	}
	opts := []asynq.Option{asynq.Queue(queueName)}
	if maxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(maxRetry))
	}
	_, err = s.client.Enqueue(task, opts...)
	return err
}

func (s *MailService) SendVerificationCode(target string, subject string, code string, expireTime int) error {
	data := templates.Data{
		Subject: subject,
		Content: templates.VerificationCodeContent{
			Code:       code,
			ExpireTime: expireTime,
		},
	}
	body, err := s.engine.Render(templates.VerificationCodeHtml, data)
	if err != nil {
		return err
	}
	msg := mail.Message{
		To:      target,
		Subject: subject,
		Body:    body,
	}
	return s.pushNewMail(msg, tasks.QueueCritical, 3)
}

func (s *MailService) SendResetPassword(target string, subject string, link string, expireTime int) error {
	data := templates.Data{
		Subject: subject,
		Content: templates.ResetPasswordContent{
			Link:       link,
			ExpireTime: expireTime,
		},
	}
	body, err := s.engine.Render(templates.ResetPasswordHtml, data)
	if err != nil {
		return err
	}
	msg := mail.Message{
		To:      target,
		Subject: subject,
		Body:    body,
	}
	return s.pushNewMail(msg, tasks.QueueCritical, 3)
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
