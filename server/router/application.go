package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
)

type ApplicationRouter struct {
}

func (r *ApplicationRouter) InitApplicationRouter(router *gin.RouterGroup) {
	var applicationApi = v1.NewApplicationApi()

	applicationRouter := router.Group("/applications")
	{
		applicationRouter.GET("/", applicationApi.GetAll)
		applicationRouter.GET("/:id", applicationApi.GetOne)
		applicationRouter.POST("/", applicationApi.New)
		applicationRouter.PATCH("/:id", applicationApi.Review)
		applicationRouter.DELETE("/:id", applicationApi.Delete)
	}
}
