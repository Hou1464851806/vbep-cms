package jwt

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const TokenExpiredDuration = time.Hour * 24

// 密钥
var secret = []byte("uzcqzTjZ7iRnEowCKA88WIOy7f52I5ezScbYgEv2BXU=")

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint) (string, error) {
	claims := Claims{
		userID,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiredDuration)),
			Issuer:    "vbep-cms",
		}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseToken(tokenString string) (*Claims, error) {
	var claims = new(Claims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
