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

type LoggerFileConfig struct {
	MaxSize    int  `mapstructure:"max_size"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAge     int  `mapstructure:"max_age"`
	Compress   bool `mapstructure:"compress"`
}

type LoggerGormConfig struct {
	Level                string `mapstructure:"level"`
	Trace                bool   `mapstructure:"trace"`
	SlowThresholdMS      int    `mapstructure:"slow_threshold_ms"`
	IgnoreRecordNotFound bool   `mapstructure:"ignore_record_not_found"`
}

type LoggerConfig struct {
	ConsoleLevel string           `mapstructure:"console_level"`
	FileLevel    string           `mapstructure:"file_level"`
	File         LoggerFileConfig `mapstructure:"file"`
	Gorm         LoggerGormConfig `mapstructure:"gorm"`
}

type Config struct {
	AppName  string         `mapstructure:"app_name"`
	AppMode  string         `mapstructure:"app_mode"` // 由环境变量控制 [api worker]
	Server   ServerConfig   `mapstructure:"server"`
	Web      WebConfig      `mapstructure:"web"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	File     FileConfig     `mapstructure:"file"`
	Cron     CronConfig     `mapstructure:"cron"`
	Email    EmailConfig    `mapstructure:"email"`
}
