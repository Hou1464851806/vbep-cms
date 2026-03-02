package initialize

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"server/global"
	"server/logger"
)

func NewViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		log.Panicf("Read config error: %v\n", err)
	}
	if err := v.Unmarshal(&global.CFG); err != nil {
		log.Fatalf("Unmarshal config error: %v\n", err)
	}
	if appMode := os.Getenv("APP_MODE"); appMode != "" {
		global.CFG.AppMode = appMode
	}
	logger.Infof("Config: %#v", global.CFG)
	return v
}
