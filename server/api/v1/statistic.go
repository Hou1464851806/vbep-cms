package v1

import (
	"github.com/gin-gonic/gin"
	"server/service"
)

type StatisticApi struct {
	service *service.StatisticService
}

func NewStatisticApi() *StatisticApi {
	return &StatisticApi{service: new(service.StatisticService)}
}

func (s *StatisticApi) GetUserNum(ctx *gin.Context) {}
