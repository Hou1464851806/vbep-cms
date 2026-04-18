package logger

import "server/global"

// ====== asynq logger zap sugar ======

type asynqZapLogger struct{}

func NewAsynqZapLogger() *asynqZapLogger {
	return &asynqZapLogger{}
}

func (l *asynqZapLogger) Debug(args ...interface{}) {
	global.SLOG.Debug(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Info(args ...interface{}) {
	global.SLOG.Info(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Warn(args ...interface{}) {
	global.SLOG.Warn(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Error(args ...interface{}) {
	global.SLOG.Error(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Fatal(args ...interface{}) {
	global.SLOG.Fatal(append([]interface{}{"[asynq] "}, args...)...)
}

// ====== Server logger zap sugar ======

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
