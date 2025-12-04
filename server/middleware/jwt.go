package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"server/pkg/jwt"
)

const CTXUserID = "user_id"

func JWTAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.Request.Header.Get("token")
		if token == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "token is empty",
			})
			ctx.Abort()
			return
		}
		claims, err := jwt.ParseToken(token)
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
