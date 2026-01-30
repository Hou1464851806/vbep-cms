package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
	"server/middleware"
	"server/model"
)

type StatisticRouter struct {
}

func (r *StatisticRouter) InitStatisticRouter(router *gin.RouterGroup) {
	var statisticApi = v1.NewStatisticApi()

	statisticRouter := router.Group("/statistics")
	statisticRouter.Use(middleware.JWTAuth())
	statisticRouter.Use(middleware.AuthRole(model.Admin))
	{
		statisticRouter.GET("/user", statisticApi.GetUser)
		statisticRouter.GET("/application", statisticApi.GetApplication)
	}
}
