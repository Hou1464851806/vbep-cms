package config

const (
	DriverSmtp = "smtp"
	DriverApi  = "api"
)

type EmailConfig struct {
	Driver string     `mapstructure:"driver"`
	From   string     `mapstructure:"from"`
	Smtp   SmtpConfig `mapstructure:"smtp"`
	Api    ApiConfig  `mapstructure:"api"`
}

type SmtpConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type ApiConfig struct {
	Url string `mapstructure:"url"`
	Key string `mapstructure:"key"`
}
