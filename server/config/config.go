package config

const (
	AppModeApi    = "api"
	AppModeWorker = "worker"
)

type FileConfig struct {
	UploadLimitApplication int `mapstructure:"upload_limit_application"`
	UploadLimitDataset     int `mapstructure:"upload_limit_dataset"`
	UploadLimitDownload    int `mapstructure:"upload_limit_download"`
}

type CronConfig struct {
	Interval int `mapstructure:"interval"` // Hours
	Offset   int `mapstructure:"offset"`   // Hours
}

type WebConfig struct {
	Url string `mapstructure:"url"`
}

type Config struct {
	AppName  string         `mapstructure:"app_name"`
	AppMode  string         `mapstructure:"app_mode"` // 由环境变量控制 [api worker]
	Server   ServerConfig   `mapstructure:"server"`
	Web      WebConfig      `mapstructure:"web"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	File     FileConfig     `mapstructure:"file"`
	Cron     CronConfig     `mapstructure:"cron"`
	Email    EmailConfig    `mapstructure:"email"`
}
