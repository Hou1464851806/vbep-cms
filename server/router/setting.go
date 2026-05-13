package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type SettingRouter struct {
}

func (r *SettingRouter) InitSettingRouter(Router *gin.RouterGroup) {
	settingApi := v1.NewSettingApi()
	Router.GET("/agreement", settingApi.GetUserAgreement)

	settingRouter := Router.Group("/settings")
	settingRouter.Use(middleware.JWTAuth())
	{
		settingRouter.GET("", middleware.AuthRole(model.Admin), settingApi.GetSetting)
		settingRouter.PATCH("", middleware.AuthRole(model.SuperAdmin), settingApi.SetSetting)
	}
}
