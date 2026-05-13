package v1

import (
	"net/http"
	"server/logger"
	"server/service"

	"github.com/gin-gonic/gin"
)

type StatisticApi struct {
	service *service.StatisticService
}

func NewStatisticApi() *StatisticApi {
	return &StatisticApi{service: service.NewStatisticService()}
}

func (s *StatisticApi) GetCurrent(c *gin.Context) {
	data, err := s.service.GetLatestStats()
	if err != nil {
		logger.Errorf("failed to get latest statistics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取统计数据失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

// RefreshStats 强制刷新统计数据
func (s *StatisticApi) RefreshStats(c *gin.Context) {
	// 使用 24h 间隔，8h 偏移 (北京时间) 进行计算
	if err := s.service.GenerateStats(); err != nil {
		logger.Errorf("failed to refresh statistics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "刷新统计数据失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "统计数据已刷新"})
}
