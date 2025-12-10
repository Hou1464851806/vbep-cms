package model

import "time"

type ApplicationStatus uint

const (
	PENDING ApplicationStatus = iota
	APPROVED
	DISAPPROVED
)

type Application struct {
	UUID           string            `gorm:"primaryKey"`
	UserID         uint              `gorm:"not null"`
	User           User              `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	DatasetUUID    string            `gorm:"not null"`
	Dataset        Dataset           `gorm:"foreignKey:DatasetUUID;references:UUID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status         ApplicationStatus `gorm:"not null;default:0"`
	Reason         string            `gorm:"not null"`
	AttachmentPath []string          `gorm:"serializer:json;type:json"`
	AuditorID      *uint
	Auditor        User `gorm:"foreignKey:AuditorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AuditorResult  *string
	AuditedAt      *time.Time `gorm:"type:timestamptz"`
	Time           Base       `gorm:"embedded"`
}
