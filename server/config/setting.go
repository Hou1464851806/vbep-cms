package config

type SettingConfig struct {
	AppFile     string `mapstructure:"app_file"`
	PublicEmail string `mapstructure:"public_email"`
}
