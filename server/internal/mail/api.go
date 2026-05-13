package mail

import (
	"bytes"
	"encoding/json"
	"net/http"
	"server/logger"
)

type ApiSender struct {
	Url string
	Key string
}

func (s *ApiSender) Send(msg Message) error {
	// temp
	payload, err := json.Marshal(map[string]string{
		"to":      msg.To,
		"title":   msg.Subject,
		"content": msg.Body,
		"key":     s.Key,
	})
	if err != nil {
		logger.Errorf("failed to marshal api mail payload to=%s subject=%s: %v", msg.To, msg.Subject, err)
		return err
	}
	_, err = http.Post(s.Url, "application/json", bytes.NewBuffer(payload))
	return err
}
