package logger

import (
	"io"
	"server/global"
	"strings"

	"go.uber.org/zap/zapcore"
)

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

// ====== gin logger zap writer ======

type ginZapWriter struct {
	level zapcore.Level
}

func NewGinZapWriter(level zapcore.Level) io.Writer {
	return &ginZapWriter{level: level}
}

func (w *ginZapWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if msg == "" {
		return len(p), nil
	}

	switch w.level {
	case zapcore.DebugLevel:
		global.SLOG.Debug(msg)
	case zapcore.WarnLevel:
		global.SLOG.Warn(msg)
	case zapcore.ErrorLevel:
		global.SLOG.Error(msg)
	case zapcore.FatalLevel:
		global.SLOG.Fatal(msg)
	default:
		global.SLOG.Info(msg)
	}

	return len(p), nil
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
