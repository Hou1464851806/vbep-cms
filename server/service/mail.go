package service

import (
	"github.com/hibiken/asynq"
	"server/internal/mail"
	"server/internal/tasks"
	"server/internal/templates"
	"server/logger"
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
		logger.Errorf("failed to create mail task to=%s subject=%s: %v", msg.To, msg.Subject, err)
		return err
	}
	opts := []asynq.Option{asynq.Queue(queueName)}
	if maxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(maxRetry))
	}
	t, err := s.client.Enqueue(task, opts...)
	if err != nil {
		logger.Errorf("failed to enqueue mail task to=%s subject=%s queue=%s: %v", msg.To, msg.Subject, queueName, err)
		return err
	}
	logger.Debugf("Task info: queue=%s max_retry=%d", t.Queue, t.MaxRetry)
	return err
}

func (s *MailService) SendNewMail(target string, subject string, templateName string, content any, queueName string, maxRetry int) error {
	data := templates.Data{
		Subject: subject,
		Content: content,
	}
	body, err := s.engine.Render(templateName, data)
	if err != nil {
		return err
	}
	msg := mail.Message{
		To:      target,
		Subject: subject,
		Body:    body,
	}
	return s.pushNewMail(msg, queueName, maxRetry)
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

// SendNewAppNotification 后面的参数应该允许使用任意的数据，接收数据后解析到模板数据上（或者可以展开参数按位置进行匹配，html模板里的参数用T1 T2来占位）
func (s *MailService) SendNewAppNotification(target string, subject string, datasetTittle string, url string) error {
	data := templates.Data{
		Subject: subject,
		Content: templates.NewAppNotificationContent{
			DatasetTitle: datasetTittle,
			Url:          url,
		},
	}
	body, err := s.engine.Render(templates.NewAppNotificationHtml, data)
	if err != nil {
		return err
	}
	msg := mail.Message{
		To:      target,
		Subject: subject,
		Body:    body,
	}
	return s.pushNewMail(msg, tasks.QueueDefault, 0)
}

func (s *MailService) SendAppChangeNotification(target string, subject string, datasetTittle string, status string, url string) error {
	data := templates.Data{
		Subject: subject,
		Content: templates.AppChangeNotificationContent{
			DatasetTitle: datasetTittle,
			Status:       status,
			Url:          url,
		},
	}
	body, err := s.engine.Render(templates.AppChangeNotificationHtml, data)
	if err != nil {
		return err
	}
	msg := mail.Message{
		To:      target,
		Subject: subject,
		Body:    body,
	}
	return s.pushNewMail(msg, tasks.QueueDefault, 0)
}
