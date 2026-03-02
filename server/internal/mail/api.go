package mail

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type ApiSender struct {
	Url string
	Key string
}

func (s *ApiSender) Send(msg Message) error {
	// temp
	payload, _ := json.Marshal(map[string]string{
		"to":      msg.To,
		"title":   msg.Subject,
		"content": msg.Body,
		"key":     s.Key,
	})
	_, err := http.Post(s.Url, "application/json", bytes.NewBuffer(payload))
	return err
}
