package initialize

import (
	"context"
	"server/global"
	"server/logger"

	"github.com/redis/go-redis/v9"
)

func Redis() *redis.Client {
	redisCfg := global.CFG.Redis
	client := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Addr,
		Password: redisCfg.Password, // no password set
		DB:       redisCfg.DB,       // use default DB
	})

	// Ping 测试连接
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		logger.Fatalw("redis connect ping failed, err", "error", err)
	}

	logger.Infow("[Success] redis connect ping response", "response", pong)
	return client
}
