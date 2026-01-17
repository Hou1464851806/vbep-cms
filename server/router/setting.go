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
	settingRouter := Router.Group("settings").Use(middleware.JWTAuth())
	{
		settingRouter.GET("", middleware.AuthRole(model.Common), v1.GetSetting)
		settingRouter.PATCH("", middleware.AuthRole(model.SuperAdmin), v1.SetSetting)
	}
}
