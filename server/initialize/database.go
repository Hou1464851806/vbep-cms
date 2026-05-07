package initialize

import (
	"fmt"
	"server/global"
	"server/logger"
	"server/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const postgresMigrationLockKey int64 = 741220891971734033

func NewDB(runMigrations bool) *gorm.DB {
	dbCfg := global.CFG.Database

	var dialector gorm.Dialector
	var logMsg string
	dbType := "postgres"

	// 2. 根据配置中的 DBType 判断使用哪个数据库
	// 注意：这里假设你在 config.toml 里配置了 db_type = "sqlite" 或 "postgres"
	switch dbCfg.DBType {
	case "sqlite":
		dbType = "sqlite"
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
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.NewGormLoggerWithConfig(global.CFG.Logger.Gorm),
	})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	logger.Successf(logMsg)

	// 4. 自动迁移
	if runMigrations {
		err = migrateSchema(db, dbType)
		if err != nil {
			//logger.Errorf("Failed to migrate schema: %v", err)
			logger.Fatalf("Failed to migrate schema: %v", err)
		}

		logger.Successf("Auto migrate schema")
	} else {
		logger.Infof("Skip auto migrate schema")
	}

	return db
}

func migrateSchema(db *gorm.DB, dbType string) error {
	if dbType != "postgres" {
		return autoMigrate(db)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", postgresMigrationLockKey).Error; err != nil {
			return fmt.Errorf("failed to acquire migration lock: %w", err)
		}

		return autoMigrate(tx)
	})
}

func autoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&model.User{},
		&model.Dataset{},
		&model.File{},
		&model.Application{},
		&model.Audit{},
		&model.Setting{},
		&model.Statistic{},
		&model.License{},
	)
	if err != nil {
		return err
	}

	// Auto mitigate existing approved applications to licenses
	// This ensures that all existing approved applications have a corresponding license.
	var apps []model.Application
	db.Where("status = ?", model.APPROVED).Find(&apps)
	for _, app := range apps {
		var count int64
		db.Model(&model.License{}).Where("application_id = ?", app.ID).Count(&count)
		if count == 0 {
			db.Create(&model.License{
				DatasetID:     app.DatasetID,
				ApplicantID:   app.ApplicantID,
				ApplicationID: app.ID,
				Status:        model.LicenseNormal,
				ExpireAt:      app.ExpireAt,
			})
		}
	}

	return nil
}
