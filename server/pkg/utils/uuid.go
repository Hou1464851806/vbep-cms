package utils

import "github.com/google/uuid"

func NewV4() string {
	UUID := uuid.New()
	return UUID.String()
}
