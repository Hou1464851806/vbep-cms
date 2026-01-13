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
	settingRouter := Router.Group("setting").Use(middleware.JWTAuth(), middleware.AuthRole(model.SuperAdmin))
	{
		settingRouter.GET("", v1.GetSetting)
		settingRouter.PATCH("", v1.SetSetting)
	}
}
