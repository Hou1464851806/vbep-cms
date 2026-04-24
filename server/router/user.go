package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type UserRouter struct {
	userApi *v1.UserApi
}

func NewUserRouter(userApi *v1.UserApi) *UserRouter {
	return &UserRouter{
		userApi: userApi,
	}
}

func (u *UserRouter) InitUserRouter(router *gin.RouterGroup) {
	userApi := u.userApi
	var authApi = v1.NewAuthApi()

	userRouter := router.Group("/users")
	{
		// user
		userRouter.POST("/", userApi.CreateUser)
		userRouter.DELETE("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Common), userApi.DeleteUser)
		userRouter.PATCH("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Common), userApi.UpdateUser)
		userRouter.GET("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Common), userApi.GetUser)
		userRouter.GET("/", middleware.JWTAuth(), middleware.AuthRole(model.Admin), userApi.GetAllUsers)

		userRouter.GET("/:id/files", middleware.JWTAuth(), middleware.AuthRole(model.SuperAdmin), userApi.GetUserAllFiles)

		// captcha
		userRouter.GET("/captcha", userApi.GetCaptcha)
		userRouter.POST("/otp", userApi.SendOTP)

		// reset
		userRouter.POST("/reset/send", userApi.SendResetPassword)
		userRouter.POST("/reset/confirm", userApi.ConfirmResetPassword)

		// auth
		userRouter.POST("/login", authApi.Login)
		userRouter.POST("/refresh", authApi.Refresh)
		userRouter.GET("/logout", middleware.JWTAuth(), middleware.AuthRole(model.Common), authApi.Logout)
	}
}
