package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type ApplicationRouter struct{}

func (r *ApplicationRouter) InitApplicationRouter(router *gin.RouterGroup) {
	var applicationApi = v1.NewApplicationApi()

	applicationRouter := router.Group("/applications")
	{
		applicationRouter.GET("/", middleware.JWTAuth(), middleware.AuthRole(model.Common), applicationApi.GetAll)
		applicationRouter.GET("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Common), applicationApi.GetOne)
		applicationRouter.POST("/", middleware.JWTAuth(), middleware.AuthRole(model.Common), applicationApi.Create)
		applicationRouter.POST("/:id/approve", middleware.JWTAuth(), middleware.AuthRole(model.Admin), applicationApi.Approve)
		applicationRouter.POST("/:id/reject", middleware.JWTAuth(), middleware.AuthRole(model.Admin), applicationApi.Reject)
	}
}
