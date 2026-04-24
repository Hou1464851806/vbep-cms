package v1

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"server/global"
	"server/logger"
	"server/model"
	"server/pkg/jwt"
	"server/service"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthApi struct {
	service *service.UserService
}

func NewAuthApi() *AuthApi {
	return &AuthApi{service: new(service.UserService)}
}

type LoginParam struct {
	UserInfo string `json:"userInfo" binding:"required"`
	PWDHash  string `json:"password" binding:"required"`
}

func (a *AuthApi) Login(ctx *gin.Context) {
	var p LoginParam
	if err := ctx.ShouldBind(&p); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "请求参数错误",
		})
		return
	}

	user, err := a.service.FindAndCheckUser(p.UserInfo, p.PWDHash)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"message": "用户名或密码错误",
		})
		return
	}

	refreshAbsoluteExpiresAt := jwt.NewRefreshAbsoluteExpiry()
	accessToken, refreshToken, refreshJTI, err := jwt.GenerateTokenPair(user.ID, refreshAbsoluteExpiresAt)
	if err != nil {
		logger.Errorf("failed to generate token pair for user %s: %v", user.ID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "登录失败，请稍后重试",
		})
		return
	}
	if err := jwt.CreateRefreshSession(refreshJTI, user.ID, refreshToken, refreshAbsoluteExpiresAt); err != nil {
		logger.Errorf("failed to persist refresh session for user %s: %v", user.ID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "登录失败，请稍后重试",
		})
		return
	}

	session, err := jwt.GetRefreshSession(refreshJTI)
	if err != nil {
		logger.Errorf("failed to load refresh session for user %s: %v", user.ID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "登录失败，请稍后重试",
		})
		return
	}
	setRefreshTokenCookie(ctx, refreshToken, session)
	ctx.JSON(http.StatusOK, gin.H{
		"token":     accessToken,
		"userID":    user.ID,
		"expiresIn": int(jwt.GetAccessTokenExpiredDuration().Seconds()),
	})
}

func (a *AuthApi) Refresh(ctx *gin.Context) {
	refreshToken, err := ctx.Cookie(jwt.GetRefreshTokenCookieName())
	if err != nil || strings.TrimSpace(refreshToken) == "" {
		clearRefreshTokenCookie(ctx)
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "登录状态已失效"})
		return
	}

	claims, err := jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		clearRefreshTokenCookie(ctx)
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "登录状态已失效"})
		return
	}
	session, err := jwt.ValidateRefreshSession(claims, refreshToken)
	if err != nil {
		logger.Warnf("invalid refresh session for user %s: %v", claims.UserID, err)
		_ = jwt.DeleteRefreshSession(claims.ID)
		clearRefreshTokenCookie(ctx)
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "登录状态已失效"})
		return
	}
	if err := jwt.EnforceRefreshInterval(session); err != nil {
		ctx.Header("Retry-After", fmt.Sprintf("%d", int(jwt.GetRefreshMinInterval().Seconds())))
		ctx.JSON(http.StatusTooManyRequests, gin.H{"message": "刷新过于频繁，请稍后再试"})
		return
	}
	if err := ensureUserExists(claims.UserID); err != nil {
		_ = jwt.DeleteRefreshSession(claims.ID)
		clearRefreshTokenCookie(ctx)
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "当前用户不存在"})
		return
	}

	newAccessToken, newRefreshToken, newRefreshJTI, err := jwt.GenerateTokenPair(claims.UserID, session.AbsoluteExpiresAt)
	if err != nil {
		logger.Errorf("failed to rotate token pair for user %s: %v", claims.UserID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "刷新登录状态失败"})
		return
	}
	if err := jwt.RotateRefreshSession(claims.ID, newRefreshJTI, claims.UserID, newRefreshToken, session); err != nil {
		logger.Errorf("failed to rotate refresh session for user %s: %v", claims.UserID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "刷新登录状态失败"})
		return
	}

	rotatedSession, err := jwt.GetRefreshSession(newRefreshJTI)
	if err != nil {
		logger.Errorf("failed to load rotated refresh session for user %s: %v", claims.UserID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "刷新登录状态失败"})
		return
	}
	setRefreshTokenCookie(ctx, newRefreshToken, rotatedSession)
	ctx.JSON(http.StatusOK, gin.H{
		"token":     newAccessToken,
		"expiresIn": int(jwt.GetAccessTokenExpiredDuration().Seconds()),
	})
}

func (a *AuthApi) Logout(ctx *gin.Context) {
	tokenString := extractBearerToken(ctx.GetHeader("Authorization"))
	if tokenString != "" {
		claims, err := jwt.ParseAccessToken(tokenString)
		if err == nil {
			duration := jwt.AccessTokenTTL(claims)
			if duration > 0 {
				if err := jwt.SetAccessTokenBlacklist(claims.ID, duration); err != nil {
					logger.Errorf("redis error when blacklisting access token: %v", err)
				}
			}
		}
	}

	refreshToken, err := ctx.Cookie(jwt.GetRefreshTokenCookieName())
	if err == nil && strings.TrimSpace(refreshToken) != "" {
		if claims, parseErr := jwt.ParseRefreshToken(refreshToken); parseErr == nil {
			if delErr := jwt.DeleteRefreshSession(claims.ID); delErr != nil {
				logger.Errorf("redis error when deleting refresh session: %v", delErr)
			}
		}
	}
	clearRefreshTokenCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "注销成功"})
}

func extractBearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}

func ensureUserExists(userID string) error {
	var user model.User
	if err := global.DB.First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	if user.ID == "" {
		return errors.New("user not found")
	}
	return nil
}

func setRefreshTokenCookie(ctx *gin.Context, refreshToken string, session *jwt.RefreshSession) {
	cookieName := jwt.GetRefreshTokenCookieName()
	maxAge := int(jwt.RefreshSessionTTL(session).Seconds())
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(cookieName, refreshToken, maxAge, "/api/v1/users", "", isCookieSecure(), true)
}

func clearRefreshTokenCookie(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(jwt.GetRefreshTokenCookieName(), "", -1, "/api/v1/users", "", isCookieSecure(), true)
}

func isCookieSecure() bool {
	webURL := strings.TrimSpace(global.CFG.Web.Url)
	if webURL == "" {
		return false
	}
	parsed, err := url.Parse(webURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https"
}
