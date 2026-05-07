package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"server/internal/mail"
	"server/logger"
	"strings"

	"go.uber.org/zap"
)

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type EmailTaskHandler struct {
	Sender mail.EmailSender
}

func (h *EmailTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	taskLogger := logger.Task(ctx, t, "tasks.EmailTaskHandler.ProcessTask")
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		taskLogger.Error("failed to unmarshal task payload", zap.Error(err))
		return fmt.Errorf("failed to unmarshal: %v: %w", err, asynq.SkipRetry)
	}
	if strings.TrimSpace(p.To) == "" {
		taskLogger.Error("invalid task payload", zap.String("field", "to"))
		return fmt.Errorf("invalid payload: to is required: %w", asynq.SkipRetry)
	}
	if strings.TrimSpace(p.Subject) == "" {
		taskLogger.Error("invalid task payload", zap.String("field", "subject"))
		return fmt.Errorf("invalid payload: subject is required: %w", asynq.SkipRetry)
	}
	if strings.TrimSpace(p.Body) == "" {
		taskLogger.Error("invalid task payload", zap.String("field", "body"))
		return fmt.Errorf("invalid payload: body is required: %w", asynq.SkipRetry)
	}
	taskLogger.Debug("start sending email",
		zap.String("to", p.To),
		zap.String("subject", p.Subject),
		zap.String("sender_type", fmt.Sprintf("%T", h.Sender)),
	)
	if err := h.Sender.Send(mail.Message{
		To:      p.To,
		Subject: p.Subject,
		Body:    p.Body,
	}); err != nil {
		taskLogger.Error("failed sending email",
			zap.String("to", p.To),
			zap.String("subject", p.Subject),
			zap.Error(err),
		)
		return err
	}
	taskLogger.Info("email sent successfully",
		zap.String("to", p.To),
		zap.String("subject", p.Subject),
	)
	return nil
}
