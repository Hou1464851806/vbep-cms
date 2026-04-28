package utils

import "os"

// IsProduction 判断当前是否为生产环境
func IsProduction() bool {
	env := os.Getenv("ENV")
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	return env == "production" || env == "prod"
}
