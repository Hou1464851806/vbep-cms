package config

type Config struct {
	AppName  string         `mapstructure:"app_name"`
	LogLevel string         `mapstructure:"log_level"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}
