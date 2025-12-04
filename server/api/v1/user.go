package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/logger"
	"server/model"
	"server/service"
)

type UserApi struct {
	service *service.UserService
}

func NewUserApi() *UserApi {
	return &UserApi{service: new(service.UserService)}
}

type CreateUserParam struct {
	Email   string `json:"email" binding:"required,email"`
	PWDHash string `json:"password" binding:"required"`
}

func (u *UserApi) CreateUser(ctx *gin.Context) {
	// 校验
	var user CreateUserParam
	if err := ctx.ShouldBind(&user); err != nil {
		logger.Errorf("Failed to validate request: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	// 入库
	newUser := &model.User{Email: user.Email, PWDHash: user.PWDHash, NickName: user.Email, Role: model.RoleUser} // 昵称默认为邮箱名
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
	id := ctx.Param("id")
	ctx.String(http.StatusOK, "Delete user id: %s", id)
}

func (u *UserApi) UpdateUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.String(http.StatusOK, "Update user id: %s", id)
}

func (u *UserApi) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.String(http.StatusOK, "Get user id: %s", id)
}

func (u *UserApi) GetAllUsers(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "all users",
	})
}
