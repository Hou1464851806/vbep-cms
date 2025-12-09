package model

import (
	"time"

	"gorm.io/gorm"
)

// Base Model 包装了记录的时间信息
type Base struct {
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
