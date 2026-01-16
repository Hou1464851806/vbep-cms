package model

type Setting struct {
	Base
	TemplateFile        string `json:"template_file" gorm:"default:''"`
	PublicEmail         string `json:"public_email" gorm:"default:''"`
	TemplateDescription string `json:"template_description" gorm:"type:text;default:''"`
}
