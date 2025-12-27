package model

import "time"

type ApplicationStatus uint

const (
	PENDING ApplicationStatus = iota
	APPROVED
	REJECTED
	RETURNED
)

// Application 申请信息
// 申请的状态需要根据审批操作进行改变，用户可进行的操作也根据申请状态进行变化
type Application struct {
	Base
	ApplicantID string            `gorm:"not null;type:char(36)"`                                                             // 申请人
	Applicant   User              `gorm:"foreignKey:ApplicantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"` // 申请人用户信息
	DatasetID   string            `gorm:"not null;type:char(36)"`                                                             // 申请的目标数据集
	Dataset     Dataset           `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`   // 目标数据集具体信息
	Status      ApplicationStatus `gorm:"not null;default:0"`                                                                 // 申请的处理状态（当前时间线下的最后一个状态）
	Reason      string            `gorm:"not null"`                                                                           // 申请的理由

	Audits []Audit `gorm:"foreignKey:ApplicationID;references:ID"` // 申请的审批记录历史
	Files  []File  `gorm:"polymorphic:Owner;"`                     // 申请的附件
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
	case RETURNED:
		return "returned" // 或 "已退回"
	default:
		return "unknown"
	}
}

func (app Application) ToListItem() ApplicationListItem {
	item := ApplicationListItem{
		ID:        app.ID,
		Status:    app.Status.String(),
		Reason:    app.Reason,
		Applicant: app.Applicant.ToBrief(),
		Dataset:   app.Dataset.ToBrief(),
		CreatedAt: app.CreatedAt,
	}

	if latest := app.latestAudit(); latest != nil {
		if latest.Auditor != nil {
			auditorBrief := latest.Auditor.ToBrief()
			item.Auditor = &auditorBrief
		}
		item.Result = latest.AuditorResult
		item.AuditedAt = latest.AuditedAt
	}

	return item
}

func (app Application) latestAudit() *Audit {
	if len(app.Audits) == 0 {
		return nil
	}

	var latest *Audit
	for i := range app.Audits {
		if latest == nil || app.Audits[i].CreatedAt.After(latest.CreatedAt) {
			latest = &app.Audits[i]
		}
	}

	return latest
}
