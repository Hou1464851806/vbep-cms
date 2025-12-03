package global

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"server/config"
)

var (
	CFG  config.Config
	VP   *viper.Viper
	LOG  *zap.Logger
	SLOG *zap.SugaredLogger
	DB   *gorm.DB
)
