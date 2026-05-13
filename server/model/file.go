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
	Name string `gorm:"not null" json:"name"`
	MD5  string `gorm:"not null" json:"md5"`
	Size uint   `json:"size"`

	OwnerID   string `gorm:"not null;index" json:"owner_id"`
	OwnerType string `gorm:"not null" json:"owner_type"`
	CreatorID string `gorm:"not null;index" json:"creator_id"`
	Type      string `gorm:"default:''" json:"type"` // e.g. "form", "identity", "general"
}

func (f *File) GetPath() string {
	// 路径生成规则：uploads/{OwnerType}/{OwnerID}/{ID}
	return filepath.Join(GetDir(f.OwnerType, f.OwnerID), f.ID)
}
