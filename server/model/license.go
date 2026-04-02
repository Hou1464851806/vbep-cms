package model

import "time"

type LicenseStatus uint

const (
	LicenseNormal LicenseStatus = iota
	LicenseRevoked
)

func (s LicenseStatus) String() string {
	if s == LicenseNormal {
		return "normal"
	} else if s == LicenseRevoked {
		return "revoked"
	}
	return "unknown"
}

// License 许可信息
type License struct {
	Base
	DatasetID     string        `gorm:"not null;type:char(36)"`
	Dataset       Dataset       `gorm:"foreignKey:DatasetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ApplicantID   string        `gorm:"not null;type:char(36)"`
	Applicant     User          `gorm:"foreignKey:ApplicantID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	ApplicationID string        `gorm:"not null;type:char(36)"`
	Application   Application   `gorm:"foreignKey:ApplicationID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Status        LicenseStatus `gorm:"not null;default:0"`
	ExpireAt      *time.Time    `json:"expireAt"`
	RevokeReason  string        `gorm:"type:text"`
}

type LicenseListItem struct {
	ID            string       `json:"id"`
	Status        string       `json:"status"`
	Dataset       DatasetBrief `json:"dataset"`
	Applicant     UserBrief    `json:"applicant"`
	ApplicationID string       `json:"applicationId"`
	CreatedAt     time.Time    `json:"createdAt"`
	ExpireAt      *time.Time   `json:"expireAt"`
	RevokeReason  string       `json:"revokeReason,omitempty"`
}

func (l License) ToListItem() LicenseListItem {
	item := LicenseListItem{
		ID:            l.ID,
		Status:        l.Status.String(),
		Dataset:       l.Dataset.ToBrief(),
		Applicant:     l.Applicant.ToBrief(),
		ApplicationID: l.ApplicationID,
		CreatedAt:     l.CreatedAt,
		ExpireAt:      l.ExpireAt,
		RevokeReason:  l.RevokeReason,
	}

	// 动态检测是否时间过期
	if l.Status == LicenseNormal && l.ExpireAt != nil && l.ExpireAt.Before(time.Now()) {
		item.Status = "expired"
		item.RevokeReason = ""
	}

	return item
}
