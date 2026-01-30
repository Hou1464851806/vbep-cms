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
	Router.GET("agreement", v1.GetUserAgreement)

	settingRouter := Router.Group("settings").Use(middleware.JWTAuth())
	{
		settingRouter.GET("", middleware.AuthRole(model.Admin), v1.GetSetting)
		settingRouter.PATCH("", middleware.AuthRole(model.SuperAdmin), v1.SetSetting)
	}
}
