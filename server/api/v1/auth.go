package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"server/logger"
	"server/middleware"
	"server/pkg/jwt"
	"server/service"
)

type AuthApi struct {
	service *service.UserService
}

func NewAuthApi() *AuthApi {
	return &AuthApi{service: new(service.UserService)}
}

type LoginParam struct {
	//UserID   uint   `json:"user_id" binding:"required"`
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

	// 设置Cookie
	maxAge := 3600 * 24 * 7 // 7 天的秒数
	path := "/"
	domain := "localhost" // 开发时可以是 localhost，生产环境要换成你的域名
	secure := false       // 开发时用 http，所以是 false；部署到线上用 https 时必须是 true
	httpOnly := true
	ctx.SetCookie("token", token, maxAge, path, domain, secure, httpOnly)
	// 返回
	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"token":   token,
	})
}

func (a *AuthApi) Logout(ctx *gin.Context) {
	uid, ok := ctx.Get(middleware.CTXUserID)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get jwt context",
		})
		return
	}
	userID, ok := uid.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "invalid jwt context type",
		})
		return
	}

	// Todo 主动注销JWT

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"user_id": userID,
	})
}
