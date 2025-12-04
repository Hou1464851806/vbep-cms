package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"server/logger"
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
	UserInfo string `json:"info" binding:"required"`
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
			"message": fmt.Errorf("failed to generate token: %w", err),
		})
	}
	// 返回Cookie
	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"token":   token,
	})
}

func (a *AuthApi) Logout(ctx *gin.Context) {

}
