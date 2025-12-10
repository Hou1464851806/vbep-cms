package model

type File struct {
	UUID string `gorm:"primaryKey"`
	Name string `gorm:"not null"`
	Path string `gorm:"not null"`
	MD5  string `gorm:"not null"`
	Size uint
	// belongs to
	DatasetUUID string  `gorm:"not null"`
	Dataset     Dataset `gorm:"foreignKey:DatasetUUID;references:UUID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	Time Base `gorm:"embedded"`
}
