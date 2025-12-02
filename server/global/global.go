package global

import (
	"github.com/spf13/viper"
	"server/config"
)

var (
	CFG config.Config
	VP  *viper.Viper
)
