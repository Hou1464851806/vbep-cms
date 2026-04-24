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

var secret = []byte("uzcqzTjZ7iRnEowCKA88WIOy7f52I5ezScbYgEv2BXU=")

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var ErrRefreshTooFrequent = errors.New("refresh requested too frequently")

type Claims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
	jwtv5.RegisteredClaims
}

type RefreshSession struct {
	UserID            string    `json:"user_id"`
	TokenHash         string    `json:"token_hash"`
	CreatedAt         time.Time `json:"created_at"`
	LastRefreshedAt   time.Time `json:"last_refreshed_at"`
	IdleExpiresAt     time.Time `json:"idle_expires_at"`
	AbsoluteExpiresAt time.Time `json:"absolute_expires_at"`
}

const expiryTolerance = time.Second

func GetAccessTokenExpiredDuration() time.Duration {
	duration := global.CFG.Server.AccessTokenExpiredDuration
	if duration <= 0 {
		duration = 15
	}
	return time.Duration(duration) * time.Minute
}

func GetRefreshTokenIdleTimeout() time.Duration {
	duration := global.CFG.Server.RefreshTokenIdleTimeout
	if duration <= 0 {
		duration = 7 * 24 * 60
	}
	return time.Duration(duration) * time.Minute
}

func GetRefreshTokenAbsoluteLifetime() time.Duration {
	duration := global.CFG.Server.RefreshTokenAbsoluteLife
	if duration <= 0 {
		duration = 30 * 24 * 60
	}
	return time.Duration(duration) * time.Minute
}

func GetRefreshMinInterval() time.Duration {
	seconds := global.CFG.Server.RefreshMinIntervalSeconds
	if seconds <= 0 {
		seconds = 30
	}
	return time.Duration(seconds) * time.Second
}

func GetRefreshTokenCookieName() string {
	name := strings.TrimSpace(global.CFG.Server.RefreshTokenCookieName)
	if name == "" {
		return "vbep_refresh_token"
	}
	return name
}

func NewRefreshAbsoluteExpiry() time.Time {
	return normalizeExpiry(time.Now().Add(GetRefreshTokenAbsoluteLifetime()))
}

func GenerateAccessToken(userID string, jti string) (string, error) {
	return generateToken(userID, jti, TokenTypeAccess, time.Now().Add(GetAccessTokenExpiredDuration()))
}

func GenerateRefreshToken(userID string, jti string, expiresAt time.Time) (string, error) {
	return generateToken(userID, jti, TokenTypeRefresh, normalizeExpiry(expiresAt))
}

func GenerateTokenPair(userID string, refreshExpiresAt time.Time) (accessToken string, refreshToken string, refreshJTI string, err error) {
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
	refreshToken, err = GenerateRefreshToken(userID, refreshJTI, refreshExpiresAt)
	if err != nil {
		return "", "", "", err
	}
	return accessToken, refreshToken, refreshJTI, nil
}

func ParseAccessToken(tokenString string) (*Claims, error) {
	return parseTokenOfType(tokenString, TokenTypeAccess)
}

func ParseRefreshToken(tokenString string) (*Claims, error) {
	return parseTokenOfType(tokenString, TokenTypeRefresh)
}

func SetAccessTokenBlacklist(jti string, expiration time.Duration) error {
	if strings.TrimSpace(jti) == "" || expiration <= 0 {
		return nil
	}
	ctx := context.Background()
	return global.REDIS.Set(ctx, accessBlacklistKey(jti), "1", expiration).Err()
}

func IsAccessTokenBlacklisted(jti string) bool {
	if strings.TrimSpace(jti) == "" {
		return false
	}
	ctx := context.Background()
	val, err := global.REDIS.Get(ctx, accessBlacklistKey(jti)).Result()
	return err == nil && val == "1"
}

func CreateRefreshSession(jti string, userID string, refreshToken string, absoluteExpiresAt time.Time) error {
	now := time.Now()
	session := RefreshSession{
		UserID:            userID,
		TokenHash:         hashToken(refreshToken),
		CreatedAt:         now,
		LastRefreshedAt:   now,
		IdleExpiresAt:     now.Add(GetRefreshTokenIdleTimeout()),
		AbsoluteExpiresAt: normalizeExpiry(absoluteExpiresAt),
	}
	return saveRefreshSession(jti, &session)
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
	if strings.TrimSpace(jti) == "" {
		return nil
	}
	session, err := GetRefreshSession(jti)
	if err != nil {
		return deleteRefreshSessionByKey(jti)
	}
	return deleteRefreshSessionWithIndex(jti, session.UserID)
}

func DeleteAllRefreshSessionsForUser(userID string) error {
	if strings.TrimSpace(userID) == "" {
		return nil
	}
	ctx := context.Background()
	indexKey := refreshUserSessionsKey(userID)
	jtis, err := global.REDIS.SMembers(ctx, indexKey).Result()
	if err != nil {
		return err
	}
	pipe := global.REDIS.TxPipeline()
	for _, jti := range jtis {
		pipe.Del(ctx, refreshSessionKey(jti))
	}
	pipe.Del(ctx, indexKey)
	_, err = pipe.Exec(ctx)
	return err
}

func ValidateRefreshSession(claims *Claims, refreshToken string) (*RefreshSession, error) {
	if claims == nil {
		return nil, errors.New("missing refresh claims")
	}
	session, err := GetRefreshSession(claims.ID)
	if err != nil {
		return nil, err
	}
	if session.UserID != claims.UserID {
		return nil, errors.New("refresh session user mismatch")
	}
	if session.TokenHash != hashToken(refreshToken) {
		return nil, errors.New("refresh session token mismatch")
	}
	now := time.Now()
	if now.After(session.IdleExpiresAt) {
		return nil, errors.New("refresh session idle timeout")
	}
	if now.After(session.AbsoluteExpiresAt) {
		return nil, errors.New("refresh session absolute timeout")
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(now) || !sameExpiry(claims.ExpiresAt.Time, session.AbsoluteExpiresAt) {
		return nil, errors.New("refresh token expiry mismatch")
	}
	return session, nil
}

func EnforceRefreshInterval(session *RefreshSession) error {
	if session == nil {
		return errors.New("missing refresh session")
	}
	if time.Since(session.LastRefreshedAt) < GetRefreshMinInterval() {
		return ErrRefreshTooFrequent
	}
	return nil
}

func RotateRefreshSession(oldJTI string, newJTI string, userID string, refreshToken string, session *RefreshSession) error {
	if session == nil {
		return errors.New("missing refresh session")
	}
	now := time.Now()
	rotated := RefreshSession{
		UserID:            userID,
		TokenHash:         hashToken(refreshToken),
		CreatedAt:         session.CreatedAt,
		LastRefreshedAt:   now,
		IdleExpiresAt:     now.Add(GetRefreshTokenIdleTimeout()),
		AbsoluteExpiresAt: normalizeExpiry(session.AbsoluteExpiresAt),
	}

	ctx := context.Background()
	ttl := RefreshSessionTTL(&rotated)
	if ttl <= 0 {
		return errors.New("refresh session expired")
	}
	payload, err := json.Marshal(rotated)
	if err != nil {
		return err
	}
	indexKey := refreshUserSessionsKey(userID)
	pipe := global.REDIS.TxPipeline()
	pipe.Set(ctx, refreshSessionKey(newJTI), payload, ttl)
	pipe.SAdd(ctx, indexKey, newJTI)
	pipe.Expire(ctx, indexKey, ttl)
	pipe.Del(ctx, refreshSessionKey(oldJTI))
	pipe.SRem(ctx, indexKey, oldJTI)
	_, err = pipe.Exec(ctx)
	return err
}

func GenerateTokenID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func AccessTokenTTL(claims *Claims) time.Duration {
	if claims == nil || claims.ExpiresAt == nil {
		return 0
	}
	return time.Until(claims.ExpiresAt.Time)
}

func RefreshSessionTTL(session *RefreshSession) time.Duration {
	if session == nil {
		return 0
	}
	expiresAt := session.AbsoluteExpiresAt
	if session.IdleExpiresAt.Before(expiresAt) {
		expiresAt = session.IdleExpiresAt
	}
	return time.Until(expiresAt)
}

func RefreshTokenExpiresAt(session *RefreshSession) time.Time {
	if session == nil {
		return time.Time{}
	}
	if session.IdleExpiresAt.Before(session.AbsoluteExpiresAt) {
		return session.IdleExpiresAt
	}
	return session.AbsoluteExpiresAt
}

func generateToken(userID string, jti string, tokenType string, expiresAt time.Time) (string, error) {
	issuedAt := time.Now()
	expiresAt = normalizeExpiry(expiresAt)
	claims := Claims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwtv5.NewNumericDate(expiresAt),
			IssuedAt:  jwtv5.NewNumericDate(issuedAt),
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

func saveRefreshSession(jti string, session *RefreshSession) error {
	if session == nil {
		return errors.New("missing refresh session")
	}
	ctx := context.Background()
	ttl := RefreshSessionTTL(session)
	if ttl <= 0 {
		return errors.New("refresh session expired")
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	indexKey := refreshUserSessionsKey(session.UserID)
	pipe := global.REDIS.TxPipeline()
	pipe.Set(ctx, refreshSessionKey(jti), payload, ttl)
	pipe.SAdd(ctx, indexKey, jti)
	pipe.Expire(ctx, indexKey, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func deleteRefreshSessionByKey(jti string) error {
	ctx := context.Background()
	return global.REDIS.Del(ctx, refreshSessionKey(jti)).Err()
}

func deleteRefreshSessionWithIndex(jti string, userID string) error {
	ctx := context.Background()
	pipe := global.REDIS.TxPipeline()
	pipe.Del(ctx, refreshSessionKey(jti))
	if strings.TrimSpace(userID) != "" {
		pipe.SRem(ctx, refreshUserSessionsKey(userID), jti)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func refreshSessionKey(jti string) string {
	return "refresh:" + jti
}

func refreshUserSessionsKey(userID string) string {
	return "refresh_user:" + userID
}

func accessBlacklistKey(jti string) string {
	return "access_blacklist:" + jti
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeExpiry(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC().Truncate(time.Second)
}

func sameExpiry(a time.Time, b time.Time) bool {
	a = normalizeExpiry(a)
	b = normalizeExpiry(b)
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	return diff <= expiryTolerance
}
