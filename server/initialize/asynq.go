package initialize

import (
	"github.com/hibiken/asynq"
	"server/global"
	"server/internal/mail"
	"server/internal/tasks"
	"server/logger"
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
		logger.Fatalf("Failed to create email sender: %v", err)
	}
	handler := &tasks.EmailTaskHandler{Sender: sender}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     global.CFG.Redis.Addr,
			Password: global.CFG.Redis.Password,
			DB:       global.CFG.Redis.DB},
		asynq.Config{
			Concurrency: 10,
		},
	)
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeEmailDelivery, handler.ProcessTask)
	if err := srv.Run(mux); err != nil {
		logger.Fatalf("Failed to start asynq worker server: %v", err)
	}
}
