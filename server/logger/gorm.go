package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
)

const slowQueryThreshold = 200 * time.Millisecond

type gormZapLogger struct {
	level gormlogger.LogLevel
}

func NewGormLogger() gormlogger.Interface {
	level := gormlogger.Warn
	if os.Getenv("LOGGER_MODE") == "debug" {
		level = gormlogger.Info
	}
	return &gormZapLogger{level: level}
}

func (l *gormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

func (l *gormZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Info {
		return
	}
	Gorm().Info(fmt.Sprintf(msg, data...))
}

func (l *gormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Warn {
		return
	}
	Gorm().Warn(fmt.Sprintf(msg, data...))
}

func (l *gormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Error {
		return
	}
	Gorm().Error(fmt.Sprintf(msg, data...))
}

func (l *gormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Float64("elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
	}
	if rows >= 0 {
		fields = append(fields, zap.Int64("rows", rows))
	}

	switch {
	case err != nil && l.level >= gormlogger.Error && (!errors.Is(err, gormlogger.ErrRecordNotFound)):
		Gorm().Error("gorm query failed", append(fields, zap.Error(err))...)
	case elapsed > slowQueryThreshold && l.level >= gormlogger.Warn:
		Gorm().Warn("gorm slow query", append(fields, zap.Duration("slow_threshold", slowQueryThreshold))...)
	case l.level >= gormlogger.Info:
		Gorm().Debug("gorm query completed", fields...)
	}
}
