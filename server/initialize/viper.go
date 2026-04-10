package initialize

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"server/global"
	"server/logger"
	"strings"
)

func getSecrets(filePath string, fallback string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Read secrets file error: %v\n", err)
		return fallback
	}
	return strings.TrimSpace(string(content))
}

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
	// 从环境变量获取部分配置
	if appMode := os.Getenv("APP_MODE"); appMode != "" {
		global.CFG.AppMode = appMode
	}
	if webUrl := os.Getenv("WEB_URL"); webUrl != "" {
		global.CFG.Web.Url = webUrl
	}
	// 用docker compose secrets覆盖数据库敏感配置
	global.CFG.Database.User = getSecrets("/run/secrets/db_user", global.CFG.Database.User)
	global.CFG.Database.Password = getSecrets("/run/secrets/db_password", global.CFG.Database.Password)
	// debug
	logger.Debugf("Config: %#v", global.CFG)
	return v
}
