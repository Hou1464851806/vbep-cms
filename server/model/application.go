package model

import "time"

type ApplicationStatus uint

const (
	PENDING ApplicationStatus = iota
	APPROVED
	REJECTED
	RETURNED
)

type ActionType uint

const (
	ActionSubmit ActionType = iota
	ActionApprove
	ActionReturn
	ActionReject
	ActionReSubmit
)

func (a ActionType) String() string {
	switch a {
	case ActionSubmit:
		return "submit"
	case ActionApprove:
		return "approve"
	case ActionReturn:
		return "return"
	case ActionReject:
		return "reject"
	case ActionReSubmit:
		return "resubmit"
	default:
		return "unknown"
	}
}

// Application 申请信息
// 申请的状态需要根据审批操作进行改变，用户可进行的操作也根据申请状态进行变化
type Application struct {
	Base
	ApplicantID string            `gorm:"not null;type:char(36)"`                                                             // 申请人
	Applicant   User              `gorm:"foreignKey:ApplicantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"` // 申请人用户信息
	DatasetID   string            `gorm:"not null;type:char(36)"`                                                             // 申请的目标数据集
	Dataset     Dataset           `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`   // 目标数据集具体信息
	Status      ApplicationStatus `gorm:"not null;default:0"`                                                                 // 申请的处理状态（当前时间线下的最后一个状态）
	Version     int               `gorm:"not null;default:0"`                                                                 // 申请的版本号，每次重新提交+1

	Audits []Audit `gorm:"foreignKey:ApplicationID;references:ID"` // 申请的审批记录历史
	Files  []File  `gorm:"polymorphic:Owner;"`                     // 申请的附件
}

// Audit History 用于记录申请的历史
// 1. 用户第一次申请提交后，需要产生一条为SUBMIT的审批记录，申请状态为pending
// 2. 审批通过，需要记录通过时的审批信息，并且同步申请状态为approved
// 3. 审批拒绝，记录此时的审批信息，同步申请状态为rejected，此操作类似于issue的close（暂无reopen）
// 4. 审批退回，记录审批信息，同步申请状态为returned，用户可以再次提交申请
// 5. 申请从“被退回”状态被再次提交时，需要产生一条为SUBMIT的审批记录，申请状态回到pending，回到开始状态
type Audit struct {
	Base
	Action        ActionType `gorm:"not null;default:0"`                                                               // 审批操作
	AuditorID     *string    `gorm:"type:char(36)"`                                                                    // 审批者
	Auditor       *User      `gorm:"foreignKey:AuditorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"` // 审批者的具体信息
	AuditorResult *string    // 审批结果文本描述

	ApplicationID string      `gorm:"not null;type:char(36)"`                                                             // 审批属于的申请ID
	Application   Application `gorm:"foreignKey:ApplicationID;reference:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // 审批历史属于申请，删除申请应该级联删除审批历史
	Files         []File      `gorm:"many2many:audit_files;"`                                                             // 该次审批/提交关联的文件快照
}

type ApplicationListItem struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	Version   int          `json:"version"`
	Applicant UserBrief    `json:"applicant"`
	Dataset   DatasetBrief `json:"dataset"`
	Auditor   *UserBrief   `json:"auditor,omitempty"`
	Result    *string      `json:"auditorResult"`
	UpdatedAt *time.Time   `json:"updatedAt"`
	CreatedAt time.Time    `json:"createdAt"`
}

type ApplicationDetail struct {
	ApplicationListItem
	Audits []AuditListItem `json:"audits"`
}

type AuditListItem struct {
	ID        string     `json:"id"`
	Action    string     `json:"action"`
	Result    *string    `json:"result,omitempty"`
	Operator  *UserBrief `json:"operator,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type AuditDetail struct {
	AuditListItem
	Files []File `json:"files"`
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
		Version:   app.Version,
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
		item.UpdatedAt = &latest.UpdatedAt
	}

	return item
}

func (app Application) ToDetail() ApplicationDetail {
	item := app.ToListItem()
	audits := make([]AuditListItem, len(app.Audits))
	for i, audit := range app.Audits {
		audits[i] = audit.ToListItem()
	}
	return ApplicationDetail{
		ApplicationListItem: item,
		Audits:              audits,
	}
}

func (a Audit) ToListItem() AuditListItem {
	item := AuditListItem{
		ID:        a.ID,
		Action:    a.Action.String(),
		Result:    a.AuditorResult,
		CreatedAt: a.CreatedAt,
		UpdatedAt: &a.UpdatedAt,
	}

	switch a.Action {
	case ActionSubmit, ActionReSubmit:
		if a.Application.Applicant.ID != "" {
			operator := a.Application.Applicant.ToBrief()
			item.Operator = &operator
		}
	default:
		if a.Auditor != nil {
			operator := a.Auditor.ToBrief()
			item.Operator = &operator
		}
	}

	return item
}

func (a Audit) ToDetail() AuditDetail {
	return AuditDetail{
		AuditListItem: a.ToListItem(),
		Files:         a.Files,
	}
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
