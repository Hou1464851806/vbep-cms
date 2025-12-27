package model

import "time"

type ActionType uint

const (
	ActionSubmit ActionType = iota
	ActionApprove
	ActionReturn
	ActionReject
	ActionReSubmit // 还是加一个操作，用于和初次提交区分
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
	AuditedAt     *time.Time // 审批的时间记录

	ApplicationID string      `gorm:"not null;type:char(36)"`                                                             // 审批属于的申请ID
	Application   Application `gorm:"foreignKey:ApplicationID;reference:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // 审批历史属于申请，删除申请应该级联删除审批历史
}

type AuditListItem struct {
	ID        string     `json:"id"`
	Action    string     `json:"action"`
	Result    *string    `json:"result,omitempty"`
	Operator  *UserBrief `json:"operator,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	AuditedAt *time.Time `json:"auditedAt,omitempty"`
}

func (a Audit) ToListItem() AuditListItem {
	item := AuditListItem{
		ID:        a.ID,
		Action:    a.Action.String(),
		Result:    a.AuditorResult,
		CreatedAt: a.CreatedAt,
		AuditedAt: a.AuditedAt,
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
