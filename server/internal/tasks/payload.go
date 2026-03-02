package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"server/internal/mail"
)

const (
	TypeEmailDelivery = "email:deliver"
)

type EmailDeliveryPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func NewEmailDeliveryTask(msg mail.Message) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailDeliveryPayload{
		To:      msg.To,
		Subject: msg.Subject,
		Body:    msg.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %s", err)
	}
	task := asynq.NewTask(TypeEmailDelivery, payload)
	return task, nil
}
