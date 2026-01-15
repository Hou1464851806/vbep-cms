package service

import (
	"server/global"
	"server/model"
)

// GetSetting retrieves the current system settings.
// It reads from the database.
func GetSetting() (*model.Setting, error) {
	var setting model.Setting
	// Use FirstOrCreate as requested.
	// Note: This will try to find a record matching the empty struct (all fields zero/empty).
	if err := global.DB.FirstOrCreate(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

// SetSetting updates the system settings.
// It updates the database record.
func SetSetting(settingsMap map[string]interface{}) error {
	// Ensure the record exists by calling GetSetting
	setting, err := GetSetting()
	if err != nil {
		return err
	}

	return global.DB.Model(setting).Updates(settingsMap).Error
}
