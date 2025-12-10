package model

type File struct {
	Base
	Name      string `gorm:"not null"`
	Path      string `gorm:"not null"`
	MD5       string `gorm:"not null"`
	Size      uint
	DatasetID string  `gorm:"not null;index"`
	Dataset   Dataset `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
