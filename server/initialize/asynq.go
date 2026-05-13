package initialize

import (
	"context"
	"github.com/hibiken/asynq"
	"server/global"
	"server/internal/mail"
	"server/internal/tasks"
	"server/logger"

	"go.uber.org/zap"
)

func Asynq() *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr:     global.CFG.Redis.Addr,
		Password: global.CFG.Redis.Password,
		DB:       global.CFG.Redis.DB,
	})
}

func StartAsynqWorkerServer() {
	sender, err := mail.NewEmailSender()
	if err != nil {
		logger.Fatalw("Failed to create email sender", "error", err)
	}
	handler := &tasks.EmailTaskHandler{Sender: sender}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     global.CFG.Redis.Addr,
			Password: global.CFG.Redis.Password,
			DB:       global.CFG.Redis.DB},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
			},
			Logger: logger.NewAsynqZapLogger(),
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Task(ctx, task, "asynq.error_handler").Error(
					"task execution failed",
					zap.Error(err),
				)
			}),
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeEmailDelivery, handler.ProcessTask)
	if err := srv.Run(mux); err != nil {
		logger.Fatalw("Failed to start asynq worker server", "error", err)
	}
}
