package model

import (
	"path/filepath"
)

const UploadRoot = "uploads"

func GetDir(ownerType, ownerID string) string {
	return filepath.Join(UploadRoot, ownerType, ownerID)
}

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
	return filepath.Join(GetDir(f.OwnerType, f.OwnerID), f.ID)
}
