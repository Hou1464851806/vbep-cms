package v1

import (
	"errors"
	"net/http"
	"server/global"
	"server/logger"
	"server/model"
	"server/service"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserApi struct {
	service *service.UserService
}

func NewUserApi() *UserApi {
	return &UserApi{service: new(service.UserService)}
}

type CreateUserParam struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required"`
	NickName string `json:"nickname"`
}

func (u *UserApi) CreateUser(ctx *gin.Context) {
	// 1. 绑定参数
	var param CreateUserParam
	if err := ctx.ShouldBind(&param); err != nil {
		logger.Errorf("Failed to validate request: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	// 2. 逻辑校验：Email 和 Phone 至少要有一个
	param.Email = strings.TrimSpace(param.Email)
	param.Phone = strings.TrimSpace(param.Phone)
	param.NickName = strings.TrimSpace(param.NickName)
	if param.Email == "" && param.Phone == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Either email or phone must be provided"})
		return
	}

	// 3. 昵称处理逻辑
	// 如果没有提供 NickName，则将其设置为 Phone 和 Email 有的那一个
	finalNickName := param.NickName
	if finalNickName == "" {
		if param.Email != "" {
			finalNickName = param.Email
		} else {
			finalNickName = param.Phone
		}
	}

	// 4. 构建 Model 对象 (入库)
	newUser := &model.User{
		PWDHash:  param.Password,
		NickName: finalNickName,
		Role:     model.Common,
	}
	if param.Email != "" {
		newUser.Email = &param.Email
	}
	if param.Phone != "" {
		newUser.Phone = &param.Phone
	}

	err := u.service.CreateUser(newUser)
	if err != nil {
		logger.Errorf("Failed to create new user in db: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	// 返回user id
	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"user_id": newUser.ID,
	})
}

func (u *UserApi) DeleteUser(ctx *gin.Context) {
	// 1. 获取 URL 参数中的目标 ID
	targetID := ctx.Param("id")
	if targetID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "User ID is required"})
		return
	}

	// 2. 获取当前请求发起人的信息
	// 这些信息是由 AuthRole 中间件放入上下文的
	obj, exists := ctx.Get(global.CTXUserObj)
	if !exists {
		// 理论上中间件已经拦截了，但为了防御性编程加上此判断
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "User not authenticated"})
		return
	}

	currentUser, ok := obj.(*model.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user info"})
		return
	}

	// 3. 权限判断逻辑
	// 规则：
	// A. 如果是管理员 (Role >= Admin)，允许删除任意人
	// B. 如果是普通用户，只能删除自己 (currentUser.ID == targetID)
	// C. 都不是，拒绝
	isSelf := currentUser.ID == targetID
	isAdmin := currentUser.Role >= model.Admin

	if !isSelf && !isAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{
			"message": "Permission denied: You can only delete your own account",
		})
		return
	}

	// 4. 调用 Service 执行删除
	err := u.service.DeleteUser(targetID)
	if err != nil {
		// 如果是“未找到用户”，返回 404；其他错误返回 500
		if err.Error() == "user not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
			return
		}

		logger.Errorf("Failed to delete user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Internal server error",
		})
		return
	}

	// 5. 成功响应
	ctx.JSON(http.StatusOK, gin.H{
		"message":    "User deleted successfully",
		"deleted_id": targetID,
	})
}

func (u *UserApi) UpdateUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.String(http.StatusOK, "Update user id: %s", id)
}

func (u *UserApi) GetUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	user, err := u.service.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "user not found",
			})
			return
		}
		logger.Errorf("failed to query user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to get user info",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"id":       user.ID,
			"nickname": user.NickName,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

func (u *UserApi) GetAllUsers(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "all users",
	})
}
