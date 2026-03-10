package utils

import "golang.org/x/crypto/bcrypt"

func PasswordToHash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
