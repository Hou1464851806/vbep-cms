package config

import "fmt"

type ServerConfig struct {
	IP                          string `mapstructure:"ip"`
	Port                        int    `mapstructure:"port"`
	AccessTokenExpiredDuration  int    `mapstructure:"access_token_expired_duration"`
	RefreshTokenExpiredDuration int    `mapstructure:"refresh_token_expired_duration"`
	RefreshTokenCookieName      string `mapstructure:"refresh_token_cookie_name"`
}

func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port) // ip:port
}
