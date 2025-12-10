package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type UserRouter struct {
}

func (u *UserRouter) InitUserRouter(router *gin.RouterGroup) {
	var userApi = v1.NewUserApi()
	var authApi = v1.NewAuthApi()

	userRouter := router.Group("/users")
	{
		// user
		userRouter.POST("/", userApi.CreateUser)
		userRouter.DELETE("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Common), userApi.DeleteUser)
		userRouter.PATCH("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Common), userApi.UpdateUser)
		userRouter.GET("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Common), userApi.GetUser)
		userRouter.GET("/", middleware.JWTAuth(), middleware.AuthRole(model.Admin), userApi.GetAllUsers)

		// auth
		userRouter.POST("/login", authApi.Login)
		userRouter.GET("/logout", middleware.JWTAuth(), middleware.AuthRole(model.Common), authApi.Logout)
	}
}
