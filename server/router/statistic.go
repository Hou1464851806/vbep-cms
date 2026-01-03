package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
	"server/middleware"
	"server/model"
)

type StatisticRouter struct {
}

func (s *StatisticRouter) InitStatisticRouter(router *gin.RouterGroup) {
	var statisticApi = v1.NewStatisticApi()

	statisticRouter := router.Group("/statistic")
	statisticRouter.Use(middleware.JWTAuth(), middleware.AuthRole(model.Admin))
	{
		statisticRouter.GET("/user", statisticApi.GetUserNum)
	}
}
