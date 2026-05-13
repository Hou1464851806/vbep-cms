package middleware

import (
	"net/http"
	"server/global"
	"server/logger"
	"server/model"

	"github.com/gin-gonic/gin"
)

// AuthRole 接收一个最低权限要求
func AuthRole(minRole model.UserRole) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1. 从上下文获取 UserID (由 JWTAuth 设置)
		userID, exists := ctx.Get(global.CTXUserID)
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
			ctx.Abort()
			return
		}

		// 2. 查询数据库获取最新用户信息 (确保角色是最新的)
		var user model.User
		// 注意：这里需要处理查不到用户的情况
		if err := global.DB.First(&user, "id = ?", userID).Error; err != nil {
			logger.Errorw("failed to load user in AuthRole middleware", "user_id", userID, "error", err)
			ctx.JSON(http.StatusUnauthorized, gin.H{"message": "当前用户不存在"})
			ctx.Abort()
			return
		}

		// 3. 检查权限
		// Common(0) < Admin(1) < SuperAdmin(2)
		if user.Role < minRole {
			ctx.JSON(http.StatusForbidden, gin.H{
				"message": "权限不足",
			})
			ctx.Abort()
			return
		}

		// 4. 将 User 对象存入上下文，供后续 Controller 使用，避免重复查库
		ctx.Set(global.CTXUserObj, &user)

		ctx.Next()
	}
}
