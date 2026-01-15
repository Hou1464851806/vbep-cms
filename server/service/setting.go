package service

import (
	"server/config"
	"server/global"
)

// GetSetting retrieves the current system settings.
// It reads directly from the in-memory global config.
func GetSetting() (config.SettingConfig, error) {
	return global.CFG.Runtime, nil
}

// SetSetting updates the system settings.
// It updates the viper instance, writes to the config file, and updates the in-memory global config.
func SetSetting(settingsMap map[string]interface{}) error {
	// The settingsMap contains only the fields present in the PATCH request body.
	for key, value := range settingsMap {
		// To improve security, you could add validation here to ensure `key`
		// is a known and modifiable setting before setting it in viper.
		viperKey := "runtime." + key
		global.VP.Set(viperKey, value)
	}

	// After setting all values in viper, write to file.
	if err := global.VP.WriteConfig(); err != nil {
		return err
	}

	// Force read from config file to ensure UnmarshalKey picks up the latest changes
	if err := global.VP.ReadInConfig(); err != nil {
		return err
	}

	// And finally, unmarshal the 'runtime' section back into our global config struct.
	// This ensures the in-memory config is in sync with the file.
	return global.VP.UnmarshalKey("runtime", &global.CFG.Runtime)
}
