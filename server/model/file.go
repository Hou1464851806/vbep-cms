package model

type File struct {
	Base
	Name string `gorm:"not null"`
	Path string `gorm:"not null"`
	MD5  string `gorm:"not null"`
	Size uint

	OwnerID   string `gorm:"not null;index"`
	OwnerType string `gorm:"not null"`
}
