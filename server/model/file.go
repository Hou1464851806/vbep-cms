package model

import (
	"path"
)

type File struct {
	Base
	Name string `gorm:"not null"`
	MD5  string `gorm:"not null"`
	Size uint

	OwnerID   string `gorm:"not null;index"`
	OwnerType string `gorm:"not null"`
	CreatorID string `gorm:"not null;index"`
}

func (f *File) GetPath() string {
	// 路径生成规则：uploads/{OwnerType}/{OwnerID}/{ID}
	// 物理存储时不保留扩展名，以防止服务器意外执行恶意脚本（如 .php, .sh）
	// 使用 path.Join 确保生成正斜杠路径，与之前存储在数据库中的格式一致
	return path.Join("uploads", f.OwnerType, f.OwnerID, f.ID)
}
