package model

type Setting struct {
	Base
	AppFile     string `json:"app_file" gorm:"default:''"`
	PublicEmail string `json:"public_email" gorm:"default:''"`
}
