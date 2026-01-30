package initialize

import (
	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
	"server/global"
	"server/logger"
	"server/tasks"
	"time"
)

func NewCron(db *gorm.DB) gocron.Scheduler {
	var dailyTaskRunTime uint
	cronConfig := global.CFG.Cron
	dailyTaskRunTime = uint(cronConfig.DailyTaskRunTime)
	// 调度器，以服务器本地时间为准
	s, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		logger.Fatalf("cron scheduler created failed, err: %v", err)
	}

	// 定义任务和工作规则
	task := gocron.NewTask(
		tasks.RunDailyStatistics,
		db,
	)
	jobDef := gocron.DailyJob(
		1,
		gocron.NewAtTimes(
			gocron.NewAtTime(dailyTaskRunTime, 0, 0),
		),
	)

	// 绑定任务至调度器
	j, err := s.NewJob(
		jobDef,
		task,
		gocron.WithName("DailyStatistics"),
	)
	if err != nil {
		logger.Fatalf("create daily task failed, err: %v", err)
	}

	// 启动调度器，默认非阻塞
	s.Start()
	// 调试
	nextRun, err := j.NextRun()
	logger.Successf("daily task [name:%s] created [ID:%s] [next run time:%v]", j.Name(), j.ID(), nextRun)
	return s
}
