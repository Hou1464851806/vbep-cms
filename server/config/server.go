package config

import "fmt"

type ServerConfig struct {
	IP                         string `mapstructure:"ip"`
	Port                       int    `mapstructure:"port"`
	AccessTokenExpiredDuration int    `mapstructure:"access_token_expired_duration"`
	RefreshTokenIdleTimeout    int    `mapstructure:"refresh_token_idle_timeout"`
	RefreshTokenAbsoluteLife   int    `mapstructure:"refresh_token_absolute_lifetime"`
	RefreshMinIntervalSeconds  int    `mapstructure:"refresh_min_interval_seconds"`
	RefreshTokenCookieName     string `mapstructure:"refresh_token_cookie_name"`
}

func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port) // ip:port
}
