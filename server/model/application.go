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
	ApplicantID string            `gorm:"not null;type:char(36)"` // 申请人
	Applicant   User              `gorm:"foreignKey:ApplicantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	DatasetID   string            `gorm:"not null;type:char(36)"` // 申请的目标数据集
	Dataset     Dataset           `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status      ApplicationStatus `gorm:"not null;default:0"` // 申请的处理状态
	Reason      string            `gorm:"not null"`           // 申请的理由
	//AttachmentPath []string          `gorm:"serializer:json;type:json"`
	AuditorID     *string `gorm:"type:char(36)"` // 审批者
	Auditor       *User   `gorm:"foreignKey:AuditorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AuditorResult *string // 审批结果
	AuditedAt     *time.Time
	Files         []File `gorm:"polymorphic:Owner;"` // 申请的附件
}
