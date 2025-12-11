package model

type File struct {
	Base
	Name string `gorm:"not null"`
	Path string `gorm:"not null"`
	MD5  string `gorm:"not null"`
	Size uint

	OwnerID   string `gorm:"not null;index"`
	OwnerType string `gorm:"not null"`

	//DatasetID     string      `gorm:"not null;index"`
	//Dataset       Dataset     `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	//ApplicationID string      `gorm:"not null;index"`
	//Application   Application `gorm:"foreignKey:ApplicationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
