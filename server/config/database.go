package config

type DatabaseConfig struct {
	IP   string `mapstructure:"ip"`
	Port int    `mapstructure:"port"`
}
