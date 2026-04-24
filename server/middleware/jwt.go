package middleware

import (
	"net/http"
	"server/global"
	"server/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := ctx.Request.Header.Get("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "缺少登录凭证",
			})
			ctx.Abort()
			return
		}

		// 2. 按空格分割，解析 Bearer Schema
		// 格式应该是: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "登录凭证格式错误",
			})
			ctx.Abort()
			return
		}

		// 3. 拿到真正的 Token 字符串
		tokenString := parts[1]

		if jwt.IsTokenInBlacklist(tokenString) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "登录状态已失效"})
			ctx.Abort()
			return
		}

		claims, err := jwt.ParseAccessToken(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "登录状态无效或已过期",
			})
			ctx.Abort()
			return
		}
		ctx.Set(global.CTXUserID, claims.UserID)
		ctx.Next()
	}
}
