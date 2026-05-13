package main

import (
	"fmt"
	"server/config"
	"server/core"
	"server/global"
	"server/initialize"
	"server/logger"
	"server/service"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	initializeServer()

	if global.CFG.AppMode == config.AppModeWorker {
		initialize.StartAsynqWorkerServer()
	} else {
		global.ASYNQ = initialize.Asynq()
		initializeCron()
	}
}

func initializeServer() {
	global.LOG, global.SLOG = initialize.NewLogger()
	global.VP = initialize.NewViper()
	global.DB = initialize.NewDB(global.CFG.AppMode != config.AppModeWorker)
	if global.CFG.AppMode != config.AppModeWorker {
		if err := initialize.EnsureBootstrapSuperAdmin(); err != nil {
			logger.Fatalf("Failed to bootstrap super admin: %v", err)
		}
	}
	global.REDIS = initialize.Redis()
}

func initializeCron() {
	// 初始化 Cron 定时任务
	// 使用 UTC+8 (北京时间)
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Warnf("failed to load timezone Asia/Shanghai, fallback to UTC+8: %v", err)
		loc = time.FixedZone("UTC+8", 8*3600)
	}
	c := cron.New(cron.WithLocation(loc))
	statsService := service.NewStatisticService()

	// 获取配置的间隔，默认为 24 小时
	interval := global.CFG.Cron.Interval
	if interval <= 0 {
		interval = 24
	}
	offset := global.CFG.Cron.Offset

	// 计算 Cron 表达式: 0 (offset%interval)/interval * * *
	// 确保 startHour 在 0 到 interval-1 之间 (处理负数 offset)
	startHour := offset % interval
	if startHour < 0 {
		startHour += interval
	}

	// 添加定时任务
	_, err = c.AddFunc(fmt.Sprintf("0 %d/%d * * *", startHour, interval), func() {
		if err := statsService.GenerateStats(); err != nil {
			logger.Errorf("Failed to generate statistics: %v", err)
		}
	})
	if err != nil {
		logger.Errorf("Failed to add cron job: %v", err)
	} else {
		logger.Successf("Statistics cron job created: interval_hours=%d", interval)
	}
	c.Start()

	core.RunServer()
}
