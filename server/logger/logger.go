package logger

import "server/global"

func Successf(template string, args ...interface{}) {
	global.SLOG.Infof("[Success] "+template, args...)
}

func Infof(template string, args ...interface{}) {
	global.SLOG.Infof(template, args...)
}

func Debugf(template string, args ...interface{}) {
	global.SLOG.Debugf(template, args...)
}

func Warnf(template string, args ...interface{}) {
	global.SLOG.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	global.SLOG.Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	global.SLOG.Fatalf(template, args...)
}
