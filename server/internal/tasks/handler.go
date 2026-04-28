package tasks

import (
	"server/pkg/apperr"

	"context"
	"encoding/json"

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
		return apperr.New(asynq.SkipRetry, "failed to unmarshal", "err", err)
	}
	if strings.TrimSpace(p.To) == "" {
		return apperr.New(asynq.SkipRetry, "invalid payload: to is required")
	}
	if strings.TrimSpace(p.Subject) == "" {
		return apperr.New(asynq.SkipRetry, "invalid payload: subject is required")
	}
	if strings.TrimSpace(p.Body) == "" {
		return apperr.New(asynq.SkipRetry, "invalid payload: body is required")
	}
	logger.Debugw("Start sending email: %+v, Sender", "p", p, "sender", h.Sender)
	if err := h.Sender.Send(mail.Message{
		To:      p.To,
		Subject: p.Subject,
		Body:    p.Body,
	}); err != nil {
		logger.Errorw("Failed sending email", "to", p.To, "subject", p.Subject, "error", err)
		return err
	}
	logger.Infow("Email sent successfully", "to", p.To, "subject", p.Subject)
	return nil
}
