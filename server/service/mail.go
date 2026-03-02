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

func (s *MailService) SendResetPassword(target string) error {
	// 1. 生成你的重置链接或验证码（业务逻辑）
	resetToken := "token_12345"
	body := fmt.Sprintf("您好，请点击链接重置密码：https://yourdomain.com/reset?token=%s", resetToken)
	// 2. 构造邮件 Message (发给用户)
	msg := mail.Message{
		To:      target,
		Subject: "重置您的密码",
		Body:    body,
	}
	// 3. 构造任务并推入队列
	task, err := tasks.NewEmailDeliveryTask(msg)
	if err != nil {
		return err
	}
	// 可以指定队列或高优先级，最多重试3次
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
