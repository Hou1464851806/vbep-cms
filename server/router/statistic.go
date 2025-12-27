package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
)

type StatisticRouter struct {
}

func (s *StatisticRouter) InitStatisticRouter(router *gin.RouterGroup) {
	var statisticApi = v1.NewStatisticApi()

	statisticRouter := router.Group("/statistic")
	{
		statisticRouter.GET("/user", statisticApi.GetUserNum)
	}
}
