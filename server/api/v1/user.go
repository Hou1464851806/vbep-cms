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

type UpdateUserParam struct {
	Email    *string         `json:"email"`
	Phone    *string         `json:"phone"`
	NickName *string         `json:"nickname"`
	Password *string         `json:"password"`
	Role     *model.UserRole `json:"role"`
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
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "必须提供邮箱或手机号"})
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
		"userID": newUser.ID,
	})
}

func (u *UserApi) DeleteUser(ctx *gin.Context) {
	// 1. 获取 URL 参数中的目标 ID
	targetID := ctx.Param("id")
	if targetID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "用户ID不能为空"})
		return
	}

	// 2. 获取当前请求发起人的信息
	// 这些信息是由 AuthRole 中间件放入上下文的
	obj, exists := ctx.Get(global.CTXUserObj)
	if !exists {
		// 理论上中间件已经拦截了，但为了防御性编程加上此判断
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	currentUser, ok := obj.(*model.User)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "解析用户信息失败"})
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
			"message": "权限不足：您只能注销自己的账户",
		})
		return
	}

	// 4. 调用 Service 执行删除
	err := u.service.DeleteUser(targetID)
	if err != nil {
		// 如果是“未找到用户”，返回 404；其他错误返回 500
		if err.Error() == "user not found" {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "用户不存在"})
			return
		}

		logger.Errorf("Failed to delete user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "服务器内部错误",
		})
		return
	}

	// 5. 成功响应
	ctx.JSON(http.StatusOK, gin.H{
		"deletedID": targetID,
	})
}

func (u *UserApi) UpdateUser(ctx *gin.Context) {
	targetID := ctx.Param("id")
	if targetID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "用户ID不能为空"})
		return
	}

	var param UpdateUserParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "参数错误: " + err.Error()})
		return
	}

	// 1. 获取操作者身份
	obj, _ := ctx.Get("user_obj")
	operator := obj.(*model.User)

	// 2. 基础鉴权：有没有资格碰到这个 User？
	// 规则：只有 "自己" 或 "管理员(Admin+)" 才能发起修改请求
	isSelf := operator.ID == targetID
	isManager := operator.Role >= model.Admin // Admin 或 SuperAdmin

	if !isSelf && !isManager {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	// 3. 构建更新 Map 并进行【字段级鉴权】
	updates := make(map[string]interface{})

	// --- A. 处理权限 (Role) ---
	// 这是最敏感的字段，优先检查
	if param.Role != nil {
		// 规则：只有 SuperAdmin 有资格修改 Role
		if operator.Role != model.SuperAdmin {
			ctx.JSON(http.StatusForbidden, gin.H{
				"message": "权限不足：只有超级管理员可以修改角色",
			})
			return
		}
		updates["role"] = *param.Role
	}

	// --- B. 处理普通字段 ---
	// 只要通过了基础鉴权 (是自己或Admin)，就有资格改这些
	if param.NickName != nil {
		updates["nick_name"] = strings.TrimSpace(*param.NickName)
	}

	if param.Email != nil {
		cleanEmail := strings.TrimSpace(*param.Email)
		if cleanEmail == "" {
			updates["email"] = nil
		} else {
			updates["email"] = cleanEmail
		}
	}

	if param.Phone != nil {
		cleanPhone := strings.TrimSpace(*param.Phone)
		if cleanPhone == "" {
			updates["phone"] = nil
		} else {
			updates["phone"] = cleanPhone
		}
	}

	if param.Password != nil {
		updates["pwd_hash"] = *param.Password
	}

	if len(updates) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "没有需要更新的字段"})
		return
	}

	// 4. 调用 Service 执行业务规则检查和入库
	// 需要 operator ID 来防止“自我降级”
	err := u.service.UpdateUser(operator, targetID, updates)
	if err != nil {
		// 简单的错误分类
		if strings.Contains(err.Error(), "user not found") {
			ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		} else if strings.Contains(err.Error(), "constraint") || strings.Contains(err.Error(), "operation failed") {
			// 业务规则错误 (如二选一失败、最后一人原则失败)
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		} else if strings.Contains(err.Error(), "occupied") {
			ctx.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		} else {
			logger.Errorf("UpdateUser error: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"message": "服务器内部错误"})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"userID": targetID})
}

func (u *UserApi) GetUser(ctx *gin.Context) {
	userID := ctx.Param("id")

	user, err := u.service.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "用户不存在",
			})
			return
		}
		logger.Errorf("failed to query user: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取用户信息失败",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"userID":   user.ID,
			"nickname": user.NickName,
			"email":    user.Email,
			"phone":    user.Phone,
			"role":     user.Role,
		},
	})
}

func (u *UserApi) GetAllUsers(ctx *gin.Context) {
	// 1. 调用 Service 获取数据
	users, err := u.service.GetAllUsers()
	if err != nil {
		logger.Errorf("failed to query all users: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取用户列表失败",
		})
		return
	}

	// 2. 构建响应数据列表
	// 我们需要手动构建切片，避免将密码等敏感字段返回给前端
	var userList []gin.H
	for _, user := range users {
		userList = append(userList, gin.H{
			"userID":    user.ID,
			"nickname":  user.NickName,
			"email":     user.Email,
			"phone":     user.Phone,
			"role":      user.Role,
			"createdAt": user.CreatedAt,
		})
	}

	// 3. 返回 JSON
	ctx.JSON(http.StatusOK, gin.H{
		"data": userList, // 这里直接放入数组
	})
}
