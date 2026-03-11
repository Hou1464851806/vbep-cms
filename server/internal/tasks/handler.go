package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"server/internal/mail"
	"server/logger"
)

type EmailTaskHandler struct {
	Sender mail.EmailSender
}

func (h *EmailTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal: %v: %w", err, asynq.SkipRetry)
	}

	logger.Debugf("Start sending email: %+v, Sender: %T", p, h.Sender)
	return h.Sender.Send(mail.Message{
		To:      p.To,
		Subject: p.Subject,
		Body:    p.Body,
	})
}
