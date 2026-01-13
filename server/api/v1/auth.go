package v1

import (
	"fmt"
	"net/http"
	"server/logger"
	"server/pkg/jwt"
	"server/service"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthApi struct {
	service *service.UserService
}

func NewAuthApi() *AuthApi {
	return &AuthApi{service: new(service.UserService)}
}

type LoginParam struct {
	UserInfo string `json:"user_info" binding:"required"`
	PWDHash  string `json:"password" binding:"required"`
}

func (a *AuthApi) Login(ctx *gin.Context) {
	// 校验参数
	var p LoginParam
	if err := ctx.ShouldBind(&p); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	// 验证用户信息
	user, err := a.service.FindAndCheckUser(p.UserInfo, p.PWDHash)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		return
	}
	logger.Successf("Valid user: %+v", user)
	// 生成JWT
	token, err := jwt.GenerateToken(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to generate token: %v", err),
		})
		return
	}
	// 返回
	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"token":   token,
		"user_id": user.ID,
	})
}

func (a *AuthApi) Logout(ctx *gin.Context) {
	// 1. 改为从 Header 获取 Token
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "no token in header"})
		return
	}
	parts := strings.SplitN(authHeader, " ", 2)
	tokenString := ""
	if len(parts) == 2 && parts[0] == "Bearer" {
		tokenString = parts[1]
	}

	// 2. 解析 Token 获取过期时间
	claims, err := jwt.ParseToken(tokenString)
	if err != nil {
		// 即使 Token 只有效性有问题，也告诉前端“注销成功”，让它删掉本地存储
		ctx.JSON(http.StatusOK, gin.H{"message": "logout success"})
		return
	}

	// 3. 计算剩余时间并写入 Redis (调用你新加的 pkg/jwt 方法)
	duration := time.Until(claims.ExpiresAt.Time)
	if duration > 0 {
		if err := jwt.SetTokenBlacklist(tokenString, duration); err != nil {
			logger.Errorf("redis error: %v", err)
			// 记录日志但不要阻拦用户注销
		}
	}

	// 4. 不需要 SetCookie("", -1) 了，直接返回
	ctx.JSON(http.StatusOK, gin.H{
		"message": "logout success",
	})
}
