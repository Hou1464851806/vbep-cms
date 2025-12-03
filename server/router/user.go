package router

import (
	"github.com/gin-gonic/gin"
	"server/api/v1"
)

type UserRouter struct {
}

func (u *UserRouter) InitUserRouter(router *gin.RouterGroup) {
	var userApi = new(v1.UserApi)
	userRouter := router.Group("/users")
	{
		userRouter.POST("/", userApi.CreateUser)
		userRouter.DELETE("/:id", userApi.DeleteUser)
		userRouter.PATCH("/:id", userApi.UpdateUser)
		userRouter.GET("/:id", userApi.GetUser)
		userRouter.GET("/", userApi.GetAllUsers)
	}
}
