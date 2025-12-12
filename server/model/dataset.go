package model

import "time"

type Dataset struct {
	Base
	Title       string `gorm:"not null"`
	Description string

	CreatorID string `gorm:"not null;type:char(36)"`
	Creator   User   `gorm:"foreignKey:CreatorID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	Files []File `gorm:"polymorphic:Owner;"`
}

type DatasetListItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Creator   string    `json:"creator"`
	FileCount int64     `json:"fileCount"`
	CreatedAt time.Time `json:"created_at"`
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
