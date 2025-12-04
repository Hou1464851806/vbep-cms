package router

import (
	"github.com/gin-gonic/gin"
	"server/api/v1"
)

type UserRouter struct {
}

func (u *UserRouter) InitUserRouter(router *gin.RouterGroup) {
	var userApi = v1.NewUserApi()
	var authApi = new(v1.AuthApi)

	userRouter := router.Group("/users")
	{
		// user
		userRouter.POST("/", userApi.CreateUser)
		userRouter.DELETE("/:id", userApi.DeleteUser)
		userRouter.PATCH("/:id", userApi.UpdateUser)
		userRouter.GET("/:id", userApi.GetUser)
		userRouter.GET("/", userApi.GetAllUsers)

		// auth
		userRouter.POST("/login", authApi.Login)
		userRouter.POST("/logout", authApi.Logout)
	}
}
