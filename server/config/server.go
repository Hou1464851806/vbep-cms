package config

import "fmt"

type ServerConfig struct {
	IP   string `mapstructure:"ip"`
	Port int    `mapstructure:"port"`
}

func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port) // ip:port
}
