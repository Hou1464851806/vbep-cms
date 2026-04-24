package jwt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"server/global"
	"strings"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// 密钥
var secret = []byte("uzcqzTjZ7iRnEowCKA88WIOy7f52I5ezScbYgEv2BXU=")

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwtv5.RegisteredClaims
}

type RefreshSession struct {
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
}

func GetAccessTokenExpiredDuration() time.Duration {
	duration := global.CFG.Server.AccessTokenExpiredDuration
	if duration <= 0 {
		duration = 15
	}
	return time.Duration(duration) * time.Minute
}

func GetRefreshTokenExpiredDuration() time.Duration {
	duration := global.CFG.Server.RefreshTokenExpiredDuration
	if duration <= 0 {
		duration = 7 * 24 * 60
	}
	return time.Duration(duration) * time.Minute
}

func GetRefreshTokenCookieName() string {
	name := strings.TrimSpace(global.CFG.Server.RefreshTokenCookieName)
	if name == "" {
		return "vbep_refresh_token"
	}
	return name
}

func GenerateTokenPair(userID string) (accessToken string, refreshToken string, refreshJTI string, err error) {
	accessJTI, err := GenerateTokenID()
	if err != nil {
		return "", "", "", err
	}
	refreshJTI, err = GenerateTokenID()
	if err != nil {
		return "", "", "", err
	}
	accessToken, err = GenerateAccessToken(userID, accessJTI)
	if err != nil {
		return "", "", "", err
	}
	refreshToken, err = GenerateRefreshToken(userID, refreshJTI)
	if err != nil {
		return "", "", "", err
	}
	return accessToken, refreshToken, refreshJTI, nil
}

func GenerateAccessToken(userID string, jti string) (string, error) {
	return generateToken(userID, jti, TokenTypeAccess, GetAccessTokenExpiredDuration())
}

func GenerateRefreshToken(userID string, jti string) (string, error) {
	return generateToken(userID, jti, TokenTypeRefresh, GetRefreshTokenExpiredDuration())
}

func ParseAccessToken(tokenString string) (*Claims, error) {
	return parseTokenOfType(tokenString, TokenTypeAccess)
}

func ParseRefreshToken(tokenString string) (*Claims, error) {
	return parseTokenOfType(tokenString, TokenTypeRefresh)
}

func SetTokenBlacklist(tokenString string, expiration time.Duration) error {
	ctx := context.Background()
	return global.REDIS.Set(ctx, "blacklist:"+tokenString, "1", expiration).Err()
}

func IsTokenInBlacklist(tokenString string) bool {
	ctx := context.Background()
	val, err := global.REDIS.Get(ctx, "blacklist:"+tokenString).Result()
	if err == nil && val == "1" {
		return true
	}
	return false
}

func SaveRefreshSession(jti string, userID string, refreshToken string, expiration time.Duration) error {
	ctx := context.Background()
	session := RefreshSession{
		UserID:    userID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: time.Now().Add(expiration),
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return global.REDIS.Set(ctx, refreshSessionKey(jti), payload, expiration).Err()
}

func RotateRefreshSession(oldJTI string, newJTI string, userID string, refreshToken string, expiration time.Duration) error {
	ctx := context.Background()
	session := RefreshSession{
		UserID:    userID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: time.Now().Add(expiration),
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	pipe := global.REDIS.TxPipeline()
	pipe.Set(ctx, refreshSessionKey(newJTI), payload, expiration)
	pipe.Del(ctx, refreshSessionKey(oldJTI))
	_, err = pipe.Exec(ctx)
	return err
}

func GetRefreshSession(jti string) (*RefreshSession, error) {
	ctx := context.Background()
	payload, err := global.REDIS.Get(ctx, refreshSessionKey(jti)).Result()
	if err != nil {
		return nil, err
	}
	var session RefreshSession
	if err := json.Unmarshal([]byte(payload), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func DeleteRefreshSession(jti string) error {
	ctx := context.Background()
	return global.REDIS.Del(ctx, refreshSessionKey(jti)).Err()
}

func ValidateRefreshSession(claims *Claims, refreshToken string) error {
	if claims == nil {
		return errors.New("missing refresh claims")
	}
	session, err := GetRefreshSession(claims.ID)
	if err != nil {
		return err
	}
	if session.UserID != claims.UserID {
		return errors.New("refresh session user mismatch")
	}
	if session.TokenHash != hashToken(refreshToken) {
		return errors.New("refresh session token mismatch")
	}
	if time.Now().After(session.ExpiresAt) {
		return errors.New("refresh session expired")
	}
	return nil
}

func GenerateTokenID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func RefreshSessionTTL(claims *Claims) time.Duration {
	if claims == nil || claims.ExpiresAt == nil {
		return 0
	}
	return time.Until(claims.ExpiresAt.Time)
}

func generateToken(userID string, jti string, tokenType string, expiration time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(expiration)),
			Issuer:    "vbep-cms",
		},
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func parseTokenOfType(tokenString string, expectedType string) (*Claims, error) {
	var claims = new(Claims)
	token, err := jwtv5.ParseWithClaims(tokenString, claims, func(token *jwtv5.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, fmt.Errorf("invalid token type: %s", claims.TokenType)
	}
	return claims, nil
}

func refreshSessionKey(jti string) string {
	return "refresh:" + jti
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
