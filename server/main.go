package main

import (
	"fmt"
	"server/core"
	"server/global"
	"server/initialize"
	"server/service"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func main() {
	initializeServer()
	initializeCron()
}

func initializeServer() {
	global.LOG, global.SLOG = initialize.NewLogger()
	global.VP = initialize.NewViper()
	global.DB = initialize.NewDB()
	global.REDIS = initialize.Redis()
}

func initializeCron() {
	// 初始化 Cron 定时任务
	c := cron.New()
	statsService := service.NewStatisticService()

	// 获取配置的间隔，默认为 24 小时
	interval := global.CFG.Cron.Interval
	if interval <= 0 {
		interval = 24
	}

	// 添加定时任务
	_, err := c.AddFunc(fmt.Sprintf("@every %dh", interval), func() {
		if err := statsService.GenerateStats(); err != nil {
			global.LOG.Error("Failed to generate statistics", zap.Error(err))
		}
	})
	if err != nil {
		global.LOG.Error("Failed to add cron job", zap.Error(err))
	} else {
		global.LOG.Info("Statistics cron job created successfully", zap.Int("interval_hours", interval))
	}
	c.Start()

	core.RunServer()
}
