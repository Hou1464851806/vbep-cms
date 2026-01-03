package model

import "time"

type DailyStat struct {
	Base
	Date     time.Time `gorm:"index;not null"`
	UserStat UserStat  `gorm:"embedded;embeddedPrefix:user_"`
	AppStat  AppStat   `gorm:"embedded;embeddedPrefix:app_"`
}

// UserStat 用于嵌入各周期统计表模型的子模型
type UserStat struct {
	Total uint `gorm:"not null"`
	New   uint `gorm:"not null"`
}

// AppStat 用于嵌入各周期统计表模型的子模型
type AppStat struct {
	Total    uint `gorm:"not null"`
	New      uint `gorm:"not null"`
	Approved uint `gorm:"not null"`
}
