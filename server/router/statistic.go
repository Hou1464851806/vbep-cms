package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type StatisticRouter struct{}

func (s *StatisticRouter) InitStatisticRouter(Router *gin.RouterGroup) {
	statRouter := Router.Group("/statistics")
	statRouter.Use(middleware.JWTAuth())
	statApi := v1.NewStatisticApi()
	{
		statRouter.GET("", middleware.AuthRole(model.Admin), statApi.GetCurrent)
		statRouter.GET("/refresh", middleware.AuthRole(model.SuperAdmin), statApi.RefreshStats)
	}
}
