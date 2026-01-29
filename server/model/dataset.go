package model

import "time"

type Dataset struct {
	Base
	Title       string `json:"title" gorm:"not null"`
	Description string `json:"description"`

	CreatorID string `json:"creatorId" gorm:"not null;type:char(36)"`
	Creator   User   `json:"creator" gorm:"foreignKey:CreatorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	Files []File `json:"files" gorm:"polymorphic:Owner;"`
}

type DatasetListItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Creator   string    `json:"creator"`
	FileCount int64     `json:"fileCount"`
	CreatedAt time.Time `json:"createdAt"`
	HasAccess bool      `json:"hasAccess"`
}

type DatasetBrief struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (d Dataset) ToBrief() DatasetBrief {
	return DatasetBrief{
		ID:    d.ID,
		Title: d.Title,
	}
}

type DatasetStatistics struct {
	TotalApplications      int64         `json:"totalApplications"`
	TotalAudits            int64         `json:"totalAudits"`
	ApprovedCount          int64         `json:"approvedCount"`
	ApprovalRate           float64       `json:"approvalRate"`
	SubmissionDistribution map[int]int64 `json:"submissionDistribution"`
}
