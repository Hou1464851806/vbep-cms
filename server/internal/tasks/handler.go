package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"server/internal/mail"
	"server/logger"
	"strings"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type EmailTaskHandler struct {
	Sender mail.EmailSender
}

func (h *EmailTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal: %v: %w", err, asynq.SkipRetry)
	}
	if strings.TrimSpace(p.To) == "" {
		return fmt.Errorf("invalid payload: to is required: %w", asynq.SkipRetry)
	}
	if strings.TrimSpace(p.Subject) == "" {
		return fmt.Errorf("invalid payload: subject is required: %w", asynq.SkipRetry)
	}
	if strings.TrimSpace(p.Body) == "" {
		return fmt.Errorf("invalid payload: body is required: %w", asynq.SkipRetry)
	}
	logger.Debugf("Start sending email: %+v, Sender: %T", p, h.Sender)
	if err := h.Sender.Send(mail.Message{
		To:      p.To,
		Subject: p.Subject,
		Body:    p.Body,
	}); err != nil {
		logger.Errorf("Failed sending email to=%s subject=%s: %v", p.To, p.Subject, err)
		return err
	}
	logger.Infof("Email sent successfully to=%s subject=%s", p.To, p.Subject)
	return nil
}
