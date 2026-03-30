package v1

import (
	"errors"
	"net/http"
	"net/url"
	"server/global"
	"server/logger"
	"server/model"
	"server/pkg/utils"
	"server/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserApi struct {
	service *service.UserService
}

func NewUserApi(userService *service.UserService) *UserApi {
	return &UserApi{service: userService}
}

type CreateUserParam struct {
	Email string `json:"email"`
	//Phone    string `json:"phone"`
	Password string `json:"password" binding:"required"`
	NickName string `json:"nickname"`
	OTP      string `json:"otp" binding:"required"`
}

type UpdateUserParam struct {
	Email *string `json:"email"`
	//Phone    *string         `json:"phone"`
	NickName *string         `json:"nickname"`
	Password *string         `json:"password"`
	Role     *model.UserRole `json:"role"`
	OTP      *string         `json:"otp"`
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

	// 2. 逻辑校验：Email 必填
	param.Email = strings.TrimSpace(param.Email)
	param.NickName = strings.TrimSpace(param.NickName)
	if param.Email == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "必须提供邮箱"})
		return
	}

	// 校验 OTP
	if !u.service.VerifyOTP(param.Email, param.OTP) {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "验证码错误或已过期"})
		return
	}

	// 密码哈希
	passwordHash, err := utils.PasswordToHash(param.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
	}

	// 3. 昵称处理逻辑
	// 如果没有提供 NickName，则将其设置为 Email
	finalNickName := param.NickName
	if finalNickName == "" {
		finalNickName = param.Email
	}

	// 4. 构建 Model 对象 (入库)
	newUser := &model.User{
		PWDHash:         passwordHash,
		NickName:        finalNickName,
		Role:            model.Common,
		Email:           param.Email,
		IsEmailVerified: true, // 注册即验证
		// Phone is optional and not provided during registration
	}

	err = u.service.CreateUser(newUser)
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
	isManager := operator.Role == model.SuperAdmin // Only SuperAdmin

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
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "邮箱不能为空"})
			return
		}
		// 验证 OTP
		if param.OTP == nil || !u.service.VerifyOTP(cleanEmail, *param.OTP) {
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "修改邮箱需要正确的验证码"})
			return
		}
		updates["email"] = cleanEmail
		updates["is_email_verified"] = true
	}

	if param.Password != nil {
		passwordHash, err := utils.PasswordToHash(*param.Password)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
		}
		updates["pwd_hash"] = passwordHash
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
			//"phone":    user.Phone,
			"role": user.Role,
		},
	})
}

func (u *UserApi) GetAllUsers(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	query := service.UserListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    ctx.Query("sort_by"),
		Order:     ctx.Query("order"),
		Role:      ctx.Query("role"),
		Keyword:   ctx.Query("keyword"),
		DateStart: ctx.Query("date_start"),
		DateEnd:   ctx.Query("date_end"),
	}
	// 1. 调用 Service 获取数据
	users, total, err := u.service.GetAllUsers(query)
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
			"userID":   user.ID,
			"nickname": user.NickName,
			"email":    user.Email,
			//"phone":     user.Phone,
			"role":      user.Role,
			"createdAt": user.CreatedAt,
		})
	}

	// 3. 返回 JSON
	ctx.JSON(http.StatusOK, gin.H{
		"data":  userList, // 这里直接放入数组
		"total": total,
	})
}

func (u *UserApi) GetCaptcha(ctx *gin.Context) {
	id, question := u.service.GenerateCaptcha()
	ctx.JSON(http.StatusOK, gin.H{
		"captchaID": id,
		"question":  question,
	})
}

type SendOTPParam struct {
	Target        string `json:"target" binding:"required"`
	CaptchaID     string `json:"captchaID" binding:"required"`
	CaptchaAnswer string `json:"captchaAnswer" binding:"required"`
}

func (u *UserApi) SendOTP(ctx *gin.Context) {
	var param SendOTPParam
	if err := ctx.ShouldBindJSON(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	if !u.service.VerifyCaptcha(param.CaptchaID, param.CaptchaAnswer) {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "验证码错误"})
		return
	}

	if err := u.service.SendOTP(param.Target); err != nil {
		logger.Errorf("Failed to send otp: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "发送失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "发送成功"})
}

type Captcha struct {
	Id     string `json:"id"`
	Answer string `json:"answer"`
}

type ResetPassReq struct {
	Email       string  `json:"email" binding:"required"`
	FrontendUrl string  `json:"frontend_url" binding:"required"`
	Captcha     Captcha `json:"captcha" binding:"required"`
}

func (u *UserApi) SendResetPassword(ctx *gin.Context) {
	var req ResetPassReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf("Failed to validate request: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	if !u.service.VerifyCaptcha(req.Captcha.Id, req.Captcha.Answer) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "验证码错误",
		})
		return
	}

	if !u.isValidLink(req.FrontendUrl) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "非法的地址",
		})
		return
	}

	err := u.service.SendResetPassword(req.Email, req.FrontendUrl)
	if err != nil {
		logger.Errorf("Failed to send reset password email: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "密码重置失败，请稍后再试",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "如果该邮箱已注册，重置链接将发送至您的邮箱",
	})
}

type ConfirmResetPassReq struct {
	NewPassword string `json:"new_password" binding:"required"`
	Token       string `json:"token" binding:"required"`
}

func (u *UserApi) ConfirmResetPassword(ctx *gin.Context) {
	var req ConfirmResetPassReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf("Failed to validate request: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	err := u.service.ConfirmResetPassword(req.Token, req.NewPassword)
	if err != nil {
		if err.Error() == "not available" {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"message": "重置链接失效或已过期，请重新申请重置密码",
			})
		} else {
			logger.Errorf("Failed to confirm new password: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": "重置密码失败，请稍后再试",
			})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func (u *UserApi) isValidLink(link string) bool {
	parsedUrl, err := url.Parse(link)
	if err != nil {
		logger.Errorf("failed to parse url: %v", err)
		return false
	}

	allowedHosts := map[string]bool{
		"localhost:5173": true,
		"127.0.0.1:8080": true,
	}

	return allowedHosts[parsedUrl.Host]
}

func (u *UserApi) GetUserAllFiles(ctx *gin.Context) {
	// 1. 从 URL 路径中获取用户ID
	userID := ctx.Param("id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "用户ID不能为空"})
		return
	}
	// 2. 从查询参数中获取分页信息
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	if err != nil || size < 1 || size > 100 { // 增加一个最大size限制，防止滥用
		size = 10
	}
	// 3. 调用 Service 层方法获取数据
	files, total, err := u.service.GetUserFiles(userID, page, size)
	if err != nil {
		// Service层已经记录了详细错误，这里只返回一个通用的服务器错误
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取文件列表失败",
		})
		return
	}
	// 4. 成功响应，返回数据和总数
	ctx.JSON(http.StatusOK, gin.H{
		"data":  files,
		"total": total,
	})
}
