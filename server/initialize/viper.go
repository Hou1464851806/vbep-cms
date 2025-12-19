package initialize

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"server/global"
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

	// 从环境变量获取数据库配置
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		log.Panicf("Read db user from env error\n")
	}
	global.CFG.Database.User = dbUser
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		log.Panicf("Read db password from env error\n")
	}
	global.CFG.Database.Password = dbPassword

	//logger.Infof("Config: %#v", global.CFG)
	return v
}
