package router

import (
	v1 "server/api/v1"
	"server/middleware"

	"github.com/gin-gonic/gin"
)

type ApplicationRouter struct{}

func (r *ApplicationRouter) InitApplicationRouter(router *gin.RouterGroup) {
	var applicationApi = v1.NewApplicationApi()

	applicationRouter := router.Group("/applications")
	{
		applicationRouter.GET("/", middleware.JWTAuth(), applicationApi.GetAll)
		applicationRouter.GET("/:id", middleware.JWTAuth(), applicationApi.GetOne)
		applicationRouter.POST("/", middleware.JWTAuth(), applicationApi.Create)
	}
}
