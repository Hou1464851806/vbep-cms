package model

type Dataset struct {
	UUID        string `gorm:"primaryKey"`
	Title       string `gorm:"not null"`
	Description string
	CreatedBy   uint `gorm:"not null"`
	// 不显示在表中，让gorm识别belongs to关系，默认引用user.id
	Creator User `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	Time Base `gorm:"embedded"`

	Files []File `gorm:"foreignKey:DatasetUUID;references:UUID"`
}
