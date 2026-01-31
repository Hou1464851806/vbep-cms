package config

type FileConfig struct {
	UploadLimitApplication int `mapstructure:"upload_limit_application"`
	UploadLimitDataset     int `mapstructure:"upload_limit_dataset"`
}

type CronConfig struct {
	Interval int `mapstructure:"interval"` // Hours
	Offset   int `mapstructure:"offset"`   // Hours
}

type Config struct {
	AppName  string         `mapstructure:"app_name"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	File     FileConfig     `mapstructure:"file"`
	Cron     CronConfig     `mapstructure:"cron"`
}
