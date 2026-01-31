package service

import (
	"server/global"
	"server/model"
)

type SettingService struct{}

func NewSettingService() *SettingService {
	return &SettingService{}
}

func (s *SettingService) GetSetting() (*model.Setting, error) {
	var setting model.Setting
	if err := global.DB.FirstOrCreate(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *SettingService) SetSetting(settingsMap map[string]interface{}) error {
	setting, err := s.GetSetting()
	if err != nil {
		return err
	}
	return global.DB.Model(setting).Updates(settingsMap).Error
}
