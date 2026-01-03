package initialize

import (
	"fmt"
	"reflect"
	"server/global"
	"server/logger"
	"server/model"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB() *gorm.DB {
	dbCfg := global.CFG.Database

	var dialector gorm.Dialector
	var logMsg string

	// 2. 根据配置中的 DBType 判断使用哪个数据库
	// 注意：这里假设你在 config.toml 里配置了 db_type = "sqlite" 或 "postgres"
	switch dbCfg.DBType {
	case "sqlite":
		// 假如未配置路径，给一个默认值
		dbPath := dbCfg.DBPath
		if dbPath == "" {
			dbPath = "database.db"
		}
		dialector = sqlite.Open(dbPath)
		logMsg = fmt.Sprintf("Connect to Postgres %s on %s", dbCfg.DBName, dbCfg.DBPath)

	case "postgres", "pgsql":
		fallthrough

	default:
		dialector = postgres.Open(dbCfg.DSN())
		logMsg = fmt.Sprintf("Connect to Postgres %s on %s:%d", dbCfg.DBName, dbCfg.Host, dbCfg.Port)
	}

	// 3. 统一建立连接
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	logger.Successf(logMsg)

	// 4. 自动迁移
	autoMigrate(db)

	return db
}

func autoMigrate(db *gorm.DB) {
	// 数据库需要迁移的模型
	models := []interface{}{
		&model.User{},
		&model.Dataset{},
		&model.File{},
		&model.Application{},
		&model.Audit{},
		&model.DailyStat{},
	}

	// 迁移
	err := db.AutoMigrate(models...)
	if err != nil {
		//logger.Errorf("Failed to migrate schema: %v", err)
		logger.Fatalf("Failed to migrate schema: %v", err)
	}

	// 通过反射打印迁移的模型名称
	modelNames := make([]string, len(models), len(models))
	for i, m := range models {
		modelName := reflect.TypeOf(m).Elem().Name()
		modelNames[i] = modelName
	}
	logger.Successf("Auto migrate schema of %d models: %s", len(models), strings.Join(modelNames, ", "))
}
