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
	applicationRouter.Use(middleware.JWTAuth())
	{
		applicationRouter.GET("/", middleware.AuthRole(model.Common), applicationApi.GetAll)
		applicationRouter.GET("/:id", middleware.AuthRole(model.Common), applicationApi.GetOne)
		applicationRouter.GET("/:id/audits/:auditID", middleware.AuthRole(model.Common), applicationApi.GetAuditDetail)
		applicationRouter.POST("/", middleware.AuthRole(model.Common), applicationApi.Create)
		applicationRouter.POST("/:id/resubmit", middleware.AuthRole(model.Common), applicationApi.Resubmit)
		applicationRouter.POST("/:id/submit", middleware.AuthRole(model.Common), applicationApi.SubmitDraft)
		applicationRouter.DELETE("/:id", middleware.AuthRole(model.Common), applicationApi.Delete)
		applicationRouter.POST("/:id/approve", middleware.AuthRole(model.Admin), applicationApi.Approve)
		applicationRouter.POST("/:id/reject", middleware.AuthRole(model.Admin), applicationApi.Reject)
		applicationRouter.POST("/:id/return", middleware.AuthRole(model.Admin), applicationApi.Return)
	}
}
