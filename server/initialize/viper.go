package initialize

import (
	"github.com/spf13/viper"
	"os"
	"server/global"
	"server/logger"
	"strings"
)

func getSecrets(filePath string, fallback string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.Errorw("Read secrets file error", "error", err)
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
		logger.Fatalw("Read config error", "error", err)
	}
	if err := v.Unmarshal(&global.CFG); err != nil {
		logger.Fatalw("Unmarshal config error", "error", err)
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
	global.CFG.Email.Smtp.Password = getSecrets("/run/secrets/smtp_password", global.CFG.Email.Smtp.Password)
	// debug
	logger.Debugw("Configv", "config", global.CFG)
	return v
}
