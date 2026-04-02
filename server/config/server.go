package config

import "fmt"

type ServerConfig struct {
	IP                   string `mapstructure:"ip"`
	Port                 int    `mapstructure:"port"`
	TokenExpiredDuration int    `mapstructure:"token_expired_duration"`
}

func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port) // ip:port
}
