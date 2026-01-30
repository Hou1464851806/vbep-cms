package tasks

import (
	"gorm.io/gorm"
	"server/logger"
	"time"
)

func RunDailyStatistics(db *gorm.DB) {
	logger.Infof("[Cron] 开始执行每日数据统计...")

	// 定时任务的执行逻辑

	time.Sleep(2 * time.Second) // 模拟计算耗时
	logger.Infof("[Cron] 每日数据统计执行完毕")
}
