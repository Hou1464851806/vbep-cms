package v1

import (
	"github.com/gin-gonic/gin"
	"server/service"
)

type StatisticApi struct {
	service *service.StatisticService
}

func NewStatisticApi() *StatisticApi {
	return &StatisticApi{service: service.NewStatisticService()}
}

func (a *StatisticApi) GetUser(ctx *gin.Context) {}

func (a StatisticApi) GetApplication(ctx *gin.Context) {}
