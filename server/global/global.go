package global

import (
	"github.com/go-co-op/gocron/v2"
	"server/config"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	CFG   config.Config
	VP    *viper.Viper
	LOG   *zap.Logger
	SLOG  *zap.SugaredLogger
	DB    *gorm.DB
	REDIS *redis.Client
	CRON  gocron.Scheduler
)

const (
	CTXUserObj = "user_obj"
	CTXUserID  = "user_id"
)
