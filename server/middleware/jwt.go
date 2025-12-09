package middleware

import (
	"fmt"
	"net/http"
	"server/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

const CTXUserID = "user_id"

func JWTAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := ctx.Request.Header.Get("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "Authorization header is missing",
			})
			ctx.Abort()
			return
		}

		// 2. 按空格分割，解析 Bearer Schema
		// 格式应该是: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "Authorization header format must be Bearer {token}",
			})
			ctx.Abort()
			return
		}

		// 3. 拿到真正的 Token 字符串
		tokenString := parts[1]

		if jwt.IsTokenInBlacklist(tokenString) {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "token invalidated"})
			ctx.Abort()
			return
		}

		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": fmt.Sprintf("failed to parse token: %v", err),
			})
			ctx.Abort()
			return
		}
		ctx.Set(CTXUserID, claims.UserID)
		ctx.Next()
	}
}
