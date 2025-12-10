package model

type Dataset struct {
	Base
	Title       string `gorm:"not null"`
	Description string

	CreatorID string `gorm:"not null;type:char(36)"`
	Creator   User   `gorm:"foreignKey:CreatorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	Files []File `gorm:"foreignKey:DatasetID;references:ID"`
}
