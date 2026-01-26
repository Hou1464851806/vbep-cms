package model

import "time"

type ApplicationStatus uint

const (
	PENDING ApplicationStatus = iota
	APPROVED
	REJECTED
	RETURNED
)

func (s ApplicationStatus) String() string {
	switch s {
	case PENDING:
		return "pending"
	case APPROVED:
		return "approved"
	case REJECTED:
		return "rejected"
	case RETURNED:
		return "returned"
	default:
		return "unknown"
	}
}

// Application 申请信息
type Application struct {
	Base
	ApplicantID string            `gorm:"not null;type:char(36)"`                                                             // 申请人
	Applicant   User              `gorm:"foreignKey:ApplicantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"` // 申请人用户信息
	DatasetID   string            `gorm:"not null;type:char(36)"`                                                             // 申请的目标数据集
	Dataset     Dataset           `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`   // 目标数据集具体信息
	Status      ApplicationStatus `gorm:"not null;default:0"`                                                                 // 申请的处理状态（当前时间线下的最后一个状态）
	Version     int               `gorm:"not null;default:0"`                                                                 // 申请的版本号，每次重新提交+1

	Audits []Audit `gorm:"foreignKey:ApplicationID;references:ID"`
}

type ApplicationListItem struct {
	ID        string       `json:"id"`
	Status    string       `json:"status"`
	Version   int          `json:"version"`
	Applicant UserBrief    `json:"applicant"`
	Dataset   DatasetBrief `json:"dataset"`
	UpdatedAt time.Time    `json:"updatedAt"`
	CreatedAt time.Time    `json:"createdAt"`
}

type ApplicationDetail struct {
	ApplicationListItem
	Audits []AuditListItem `json:"audits"`
}

func (app Application) ToListItem() ApplicationListItem {
	item := ApplicationListItem{
		ID:        app.ID,
		Status:    app.Status.String(),
		Version:   app.Version,
		Applicant: app.Applicant.ToBrief(),
		Dataset:   app.Dataset.ToBrief(),
		CreatedAt: app.CreatedAt,
		UpdatedAt: app.UpdatedAt,
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

// Audit History 用于记录申请的历史
// 1. 用户第一次申请提交后，需要产生一条为SUBMIT的审批记录，申请状态为pending
// 2. 审批通过，需要记录通过时的审批信息，并且同步申请状态为approved
// 3. 审批拒绝，记录此时的审批信息，同步申请状态为rejected，此操作类似于issue的close（暂无reopen）
// 4. 审批退回，记录审批信息，同步申请状态为returned，用户可以再次提交申请
// 5. 申请从“被退回”状态被再次提交时，需要产生一条为SUBMIT的审批记录，申请状态回到pending，回到开始状态
type Audit struct {
	Base
	Action     ApplicationStatus `gorm:"not null;default:0"`                                                                // 审批操作/状态变更
	OperatorID *string           `gorm:"type:char(36)"`                                                                     // 操作人
	Operator   *User             `gorm:"foreignKey:OperatorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"` // 操作人的具体信息
	Result     *string           // 处理结果文本描述

	ApplicationID string      `gorm:"not null;type:char(36)"`                                                             // 审批属于的申请ID
	Application   Application `gorm:"foreignKey:ApplicationID;reference:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // 审批历史属于申请，删除申请应该级联删除审批历史
	Files         []File      `gorm:"many2many:audit_files;"`                                                             // 该次审批/提交关联的文件快照
}

type AuditListItem struct {
	ID        string     `json:"id"`
	Action    string     `json:"action"`
	Result    *string    `json:"result,omitempty"`
	Operator  *UserBrief `json:"operator"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type AuditDetail struct {
	AuditListItem
	Files []File `json:"files"`
}

func (a Audit) ToListItem() AuditListItem {
	item := AuditListItem{
		ID:        a.ID,
		Action:    a.Action.String(),
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}

	if a.Operator != nil {
		operator := a.Operator.ToBrief()
		item.Operator = &operator

		result := a.Result
		item.Result = result
	}

	return item
}

func (a Audit) ToDetail() AuditDetail {
	return AuditDetail{
		AuditListItem: a.ToListItem(),
		Files:         a.Files,
	}
}
