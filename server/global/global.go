package global

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"server/config"
)

var (
	CFG  config.Config
	VP   *viper.Viper
	LOG  *zap.Logger
	SLOG *zap.SugaredLogger
)
