package model

import (
	"server/pkg/utils"
	"time"

	"gorm.io/gorm"
)

// Base Model 包装了记录的时间信息
type Base struct {
	ID        string         `gorm:"primaryKey;type:char(36)" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == "" {
		b.ID = utils.NewV4()
	}
	return
}
