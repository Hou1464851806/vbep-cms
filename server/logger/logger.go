package logger

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"server/config"
	"server/global"
)

const (
	sourceApp   = "app"
	sourceGin   = "gin"
	sourceGorm  = "gorm"
	sourceAsynq = "asynq"
)

func ServiceName() string {
	mode := config.AppModeApi
	if global.CFG.AppMode != "" {
		mode = global.CFG.AppMode
	}
	return mode
}

func Base() *zap.Logger {
	if global.LOG == nil {
		return zap.NewNop()
	}
	return global.LOG
}

func App() *zap.Logger {
	return Base().With(zap.String(fieldSource, sourceApp))
}

func Gin() *zap.Logger {
	return Base().With(
		zap.String(fieldSource, sourceGin),
		zap.String(fieldCaller, sourceGin),
	)
}

func Gorm() *zap.Logger {
	return Base().With(
		zap.String(fieldSource, sourceGorm),
		zap.String(fieldCaller, sourceGorm),
	)
}

func Asynq() *zap.Logger {
	return Base().With(
		zap.String(fieldSource, sourceAsynq),
		zap.String(fieldCaller, sourceAsynq),
	)
}

func Task(ctx context.Context, task *asynq.Task, handlerName string) *zap.Logger {
	fields := []zap.Field{
		zap.String(fieldSource, sourceAsynq),
		zap.String(fieldCaller, handlerName),
		zap.String("task_type", task.Type()),
	}
	if taskID, ok := asynq.GetTaskID(ctx); ok {
		fields = append(fields, zap.String("task_id", taskID))
	}
	if queue, ok := asynq.GetQueueName(ctx); ok {
		fields = append(fields, zap.String("queue", queue))
	}
	if retry, ok := asynq.GetRetryCount(ctx); ok {
		fields = append(fields, zap.Int("retry", retry))
	}
	if maxRetry, ok := asynq.GetMaxRetry(ctx); ok {
		fields = append(fields, zap.Int("max_retry", maxRetry))
	}
	return Base().With(fields...)
}

func WithHTTPContext(c *gin.Context) []zap.Field {
	fields := []zap.Field{
		zap.String("method", c.Request.Method),
		zap.String("path", c.FullPath()),
		zap.String("client_ip", c.ClientIP()),
	}
	if fields[1].String == "" {
		fields[1] = zap.String("path", c.Request.URL.Path)
	}
	if handler := shortHandlerName(c.HandlerName()); handler != "" {
		fields = append(fields, zap.String("handler", handler), zap.String(fieldCaller, handler))
	} else {
		fields = append(fields, zap.String(fieldCaller, sourceGin))
	}
	if userID := c.GetString(global.CTXUserID); userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}
	return fields
}

func shortHandlerName(name string) string {
	if name == "" {
		return ""
	}
	name = strings.TrimSuffix(name, "-fm")
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

func runtimeCaller(skip int) string {
	for depth := skip; depth < skip+8; depth++ {
		pc, file, line, ok := runtime.Caller(depth)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			return fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
		name := fn.Name()
		name = strings.TrimSuffix(name, "-fm")
		parts := strings.Split(name, "/")
		shortName := parts[len(parts)-1]
		if strings.HasPrefix(shortName, "logger.") {
			continue
		}
		return shortName
	}
	return sourceApp
}

func appFields(fields []zap.Field, skip int) []zap.Field {
	return append([]zap.Field{zap.String(fieldCaller, runtimeCaller(skip))}, fields...)
}

func logApp(level zapcore.Level, msg string, skip int, fields ...zap.Field) {
	allFields := appFields(fields, skip)
	switch level {
	case zapcore.DebugLevel:
		App().Debug(msg, allFields...)
	case zapcore.WarnLevel:
		App().Warn(msg, allFields...)
	case zapcore.ErrorLevel:
		App().Error(msg, allFields...)
	case zapcore.FatalLevel:
		App().Fatal(msg, allFields...)
	default:
		App().Info(msg, allFields...)
	}
}

func Success(msg string, fields ...zap.Field) {
	logApp(zapcore.InfoLevel, msg, 3, fields...)
}

func Info(msg string, fields ...zap.Field) {
	logApp(zapcore.InfoLevel, msg, 3, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	logApp(zapcore.DebugLevel, msg, 3, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	logApp(zapcore.WarnLevel, msg, 3, fields...)
}

func Error(msg string, fields ...zap.Field) {
	logApp(zapcore.ErrorLevel, msg, 3, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	logApp(zapcore.FatalLevel, msg, 3, fields...)
}

func Successf(template string, args ...interface{}) {
	logApp(zapcore.InfoLevel, fmt.Sprintf("[Success] "+template, args...), 3)
}

func Infof(template string, args ...interface{}) {
	logApp(zapcore.InfoLevel, fmt.Sprintf(template, args...), 3)
}

func Debugf(template string, args ...interface{}) {
	logApp(zapcore.DebugLevel, fmt.Sprintf(template, args...), 3)
}

func Warnf(template string, args ...interface{}) {
	logApp(zapcore.WarnLevel, fmt.Sprintf(template, args...), 3)
}

func Errorf(template string, args ...interface{}) {
	logApp(zapcore.ErrorLevel, fmt.Sprintf(template, args...), 3)
}

func Fatalf(template string, args ...interface{}) {
	logApp(zapcore.FatalLevel, fmt.Sprintf(template, args...), 3)
}

type asynqZapLogger struct{}

func NewAsynqZapLogger() *asynqZapLogger {
	return &asynqZapLogger{}
}

func (l *asynqZapLogger) Debug(args ...interface{}) {
	Asynq().Debug(fmt.Sprint(args...))
}

func (l *asynqZapLogger) Info(args ...interface{}) {
	Asynq().Info(fmt.Sprint(args...))
}

func (l *asynqZapLogger) Warn(args ...interface{}) {
	Asynq().Warn(fmt.Sprint(args...))
}

func (l *asynqZapLogger) Error(args ...interface{}) {
	Asynq().Error(fmt.Sprint(args...))
}

func (l *asynqZapLogger) Fatal(args ...interface{}) {
	Asynq().Fatal(fmt.Sprint(args...))
}
