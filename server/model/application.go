package model

import "time"

type ApplicationStatus uint

const (
	PENDING ApplicationStatus = iota
	APPROVED
	REJECTED
)

type Application struct {
	Base
	ApplicantID   string            `gorm:"not null;type:char(36)"` // 申请人
	Applicant     User              `gorm:"foreignKey:ApplicantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	DatasetID     string            `gorm:"not null;type:char(36)"` // 申请的目标数据集
	Dataset       Dataset           `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status        ApplicationStatus `gorm:"not null;default:0"` // 申请的处理状态
	Reason        string            `gorm:"not null"`           // 申请的理由
	AuditorID     *string           `gorm:"type:char(36)"`      // 审批者
	Auditor       *User             `gorm:"foreignKey:AuditorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AuditorResult *string
	AuditedAt     *time.Time
	Files         []File `gorm:"polymorphic:Owner;"` // 申请的附件
}

type ApplicationListItem struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	Reason    string       `json:"reason"`
	Applicant UserBrief    `json:"applicant"`
	Dataset   DatasetBrief `json:"dataset"`
	Auditor   *UserBrief   `json:"auditor,omitempty"`
	Result    *string      `json:"auditorResult"`
	AuditedAt *time.Time   `json:"auditedAt"`
	CreatedAt time.Time    `json:"createdAt"`
}

func (s ApplicationStatus) String() string {
	switch s {
	case PENDING:
		return "pending" // 或 "待审批"
	case APPROVED:
		return "approved" // 或 "已通过"
	case REJECTED:
		return "rejected" // 或 "已拒绝"
	default:
		return "unknown"
	}
}
