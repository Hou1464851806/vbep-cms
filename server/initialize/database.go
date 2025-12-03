package initialize

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"server/global"
	"server/logger"
)

func NewDB() *gorm.DB {
	dbCfg := global.CFG.Database

	db, err := gorm.Open(postgres.Open(dbCfg.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	logger.Infof("[Success] connect to database %s on %s:%d", dbCfg.DBName, dbCfg.Host, dbCfg.Port)

	return db
}
