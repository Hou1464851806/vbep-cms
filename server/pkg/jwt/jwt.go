package jwt

import (
	"context"
	"errors"
	"server/global"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 密钥
var secret = []byte("uzcqzTjZ7iRnEowCKA88WIOy7f52I5ezScbYgEv2BXU=")

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GetTokenExpiredDuration() time.Duration {
	duration := global.CFG.Server.TokenExpiredDuration
	if duration <= 0 {
		duration = 30 // 默认30分钟
	}
	return time.Duration(duration) * time.Minute
}

func GenerateToken(userID string) (string, error) {
	claims := Claims{
		userID,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(GetTokenExpiredDuration())),
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

// SetTokenBlacklist 将 token 加入 Redis 黑名单
func SetTokenBlacklist(tokenString string, expiration time.Duration) error {
	ctx := context.Background()
	// key 建议加前缀，防止冲突
	return global.REDIS.Set(ctx, "blacklist:"+tokenString, "1", expiration).Err()
}

// IsTokenInBlacklist 检查 token 是否在黑名单中
func IsTokenInBlacklist(tokenString string) bool {
	ctx := context.Background()
	val, err := global.REDIS.Get(ctx, "blacklist:"+tokenString).Result()
	// 如果没有错误且值为 "1"，说明在黑名单中
	if err == nil && val == "1" {
		return true
	}
	return false
}
