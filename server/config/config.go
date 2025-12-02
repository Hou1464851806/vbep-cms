package config

type Config struct {
	AppName  string         `mapstructure:"app_name"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}
