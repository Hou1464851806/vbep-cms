package initialize

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"server/global"
	"server/logger"
	"server/model"
)

func NewDB() *gorm.DB {
	dbCfg := global.CFG.Database

	db, err := gorm.Open(postgres.Open(dbCfg.DSN()), &gorm.Config{})
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}

	logger.Successf("Connect to database %s on %s:%d", dbCfg.DBName, dbCfg.Host, dbCfg.Port)

	err = autoMigrate(db)
	if err != nil {
		logger.Errorf("Failed to migrate schema: %v", err)
	}

	logger.Successf("Auto migrate schema")

	//// test database
	//ctx := context.Background()
	//err = gorm.G[model.User](db).Create(ctx, &model.User{
	//	Role:     model.RoleAdmin,
	//	NickName: "Nonbeing",
	//	Phone:    "114514",
	//	Email:    "114514@qq.com",
	//	PWDHash:  "abcdedfg114514",
	//})
	//if err != nil {
	//	logger.Errorf("Failed to create a new user record: %v", err)
	//}

	return db
}

func autoMigrate(db *gorm.DB) error {
	err := db.AutoMigrate(&model.User{})
	return err
}
