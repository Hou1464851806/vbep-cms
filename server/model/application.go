package model

import "time"

type ApplicationStatus uint

const (
	PENDING ApplicationStatus = iota
	APPROVED
	DISAPPROVED
)

type Application struct {
	Base
	ApplicantID    string            `gorm:"not null;type:char(36)"`
	Applicant      User              `gorm:"foreignKey:ApplicantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	DatasetID      string            `gorm:"not null;type:char(36)"`
	Dataset        Dataset           `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status         ApplicationStatus `gorm:"not null;default:0"`
	Reason         string            `gorm:"not null"`
	AttachmentPath []string          `gorm:"serializer:json;type:json"`
	AuditorID      *string           `gorm:"not null;type:char(36)"`
	Auditor        User              `gorm:"foreignKey:AuditorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AuditorResult  *string
	AuditedAt      *time.Time
}
