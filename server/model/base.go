package model

import (
	"gorm.io/gorm"
	"time"
)

// Base Model 包装了记录的时间信息
type Base struct {
	CreatedAt time.Time      `gorm:"type:timestamptz"`       // 带时区的时间
	UpdatedAt time.Time      `gorm:"type:timestamptz"`       // 带时区的时间
	DeletedAt gorm.DeletedAt `gorm:"index;type:timestamptz"` // 软删除
}
