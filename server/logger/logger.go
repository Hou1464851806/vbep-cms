package logger

import (
	"context"
	"io"
	"server/global"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ====== asynq logger ======

type asynqZapLogger struct {
	logger *zap.SugaredLogger
}

var (
	asynqLogger *zap.SugaredLogger
	asynqOnce   sync.Once
)

func getAsynqLogger() *zap.SugaredLogger {
	asynqOnce.Do(func() {
		asynqLogger = global.SLOG.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	})
	return asynqLogger
}

func NewAsynqZapLogger() *asynqZapLogger {
	return &asynqZapLogger{
		logger: getAsynqLogger(),
	}
}

func (l *asynqZapLogger) Debug(args ...interface{}) {
	l.logger.Debug(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Info(args ...interface{}) {
	l.logger.Info(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Warn(args ...interface{}) {
	l.logger.Warn(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Error(args ...interface{}) {
	l.logger.Error(append([]interface{}{"[asynq] "}, args...)...)
}

func (l *asynqZapLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(append([]interface{}{"[asynq] "}, args...)...)
}

// ====== gin logger zap writer ======

type ginZapWriter struct {
	level zapcore.Level
}

var (
	ginLogger *zap.SugaredLogger
	ginOnce   sync.Once
)

func getGinLogger() *zap.SugaredLogger {
	ginOnce.Do(func() {
		// ginZapWriter.Write 增加内部调用层级
		ginLogger = global.SLOG.Desugar().WithOptions(zap.AddCallerSkip(3)).Sugar()
	})
	return ginLogger
}

func NewGinZapWriter(level zapcore.Level) io.Writer {
	return &ginZapWriter{level: level}
}

func (w *ginZapWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if msg == "" {
		return len(p), nil
	}

	l := getGinLogger()
	switch w.level {
	case zapcore.DebugLevel:
		l.Debug(msg)
	case zapcore.WarnLevel:
		l.Warn(msg)
	case zapcore.ErrorLevel:
		l.Error(msg)
	case zapcore.FatalLevel:
		l.Fatal(msg)
	default:
		l.Info(msg)
	}

	return len(p), nil
}

// ====== Server logger ======

var (
	ctxBaseLogger *zap.SugaredLogger
	ctxOnce       sync.Once
)

func getCtxBaseLogger() *zap.SugaredLogger {
	ctxOnce.Do(func() {
		// ContextLogger 内部增加了一层封装，因此统一跳过一层 Caller
		ctxBaseLogger = global.SLOG.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	})
	return ctxBaseLogger
}

type ContextLogger struct {
	logger *zap.SugaredLogger
}

func Ctx(ctx context.Context) *ContextLogger {
	l := getCtxBaseLogger()

	if ctx == nil {
		return &ContextLogger{logger: l}
	}

	var traceID, srcIP string
	var userID interface{}

	if ginCtx, ok := ctx.(*gin.Context); ok {
		traceID = ginCtx.GetString("trace_id")
		if traceID == "" {
			traceID = ginCtx.GetHeader("X-Request-Id")
		}
		if traceID == "" {
			traceID = ginCtx.GetHeader("X-Trace-Id")
		}

		if uid, exists := ginCtx.Get(global.CTXUserID); exists {
			userID = uid
		}

		srcIP = ginCtx.ClientIP()
	} else {
		if tid := ctx.Value("trace_id"); tid != nil {
			traceID, _ = tid.(string)
		}
		if uid := ctx.Value(global.CTXUserID); uid != nil {
			userID = uid
		}
		if sid := ctx.Value("src_ip"); sid != nil {
			srcIP, _ = sid.(string)
		}
	}

	var fields []interface{}
	if traceID != "" {
		fields = append(fields, "trace_id", traceID) // 在根级别输出 trace_id
	}
	if userID != nil {
		fields = append(fields, "user_id", userID)
	}
	if srcIP != "" {
		fields = append(fields, "src_ip", srcIP)
	}

	if len(fields) > 0 {
		return &ContextLogger{logger: l.With(fields...)}
	}

	return &ContextLogger{logger: l}
}

// FieldProvider defines an interface for errors that carry structured fields.
type FieldProvider interface {
	GetFields() []interface{}
}

func extractErrorFields(keysAndValues []interface{}) []interface{} {
	var newFields []interface{}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := keysAndValues[i]
			val := keysAndValues[i+1]

			if err, ok := val.(error); ok {
				curr := err
				for curr != nil {
					if fp, ok := curr.(FieldProvider); ok {
						newFields = append(newFields, fp.GetFields()...)
					}
					if unwrapper, ok := curr.(interface{ Unwrap() error }); ok {
						curr = unwrapper.Unwrap()
					} else {
						curr = nil
					}
				}
			}
			newFields = append(newFields, key, val)
		} else {
			newFields = append(newFields, keysAndValues[i])
		}
	}
	return newFields
}

// === ContextLogger Instance Methods ===
// 自动将多余的 key-value 打包进 JSON 中的 "context" 字段

func (l *ContextLogger) Debugw(msg string, keysAndValues ...interface{}) {
	keysAndValues = extractErrorFields(keysAndValues)
	if len(keysAndValues) > 0 {
		l.logger.Desugar().With(zap.Namespace("context")).Sugar().Debugw(msg, keysAndValues...)
	} else {
		l.logger.Debug(msg)
	}
}

func (l *ContextLogger) Infow(msg string, keysAndValues ...interface{}) {
	keysAndValues = extractErrorFields(keysAndValues)
	if len(keysAndValues) > 0 {
		l.logger.Desugar().With(zap.Namespace("context")).Sugar().Infow(msg, keysAndValues...)
	} else {
		l.logger.Info(msg)
	}
}

func (l *ContextLogger) Warnw(msg string, keysAndValues ...interface{}) {
	keysAndValues = extractErrorFields(keysAndValues)
	if len(keysAndValues) > 0 {
		l.logger.Desugar().With(zap.Namespace("context")).Sugar().Warnw(msg, keysAndValues...)
	} else {
		l.logger.Warn(msg)
	}
}

func (l *ContextLogger) Errorw(msg string, keysAndValues ...interface{}) {
	keysAndValues = extractErrorFields(keysAndValues)
	if len(keysAndValues) > 0 {
		l.logger.Desugar().With(zap.Namespace("context")).Sugar().Errorw(msg, keysAndValues...)
	} else {
		l.logger.Error(msg)
	}
}

func (l *ContextLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	keysAndValues = extractErrorFields(keysAndValues)
	if len(keysAndValues) > 0 {
		l.logger.Desugar().With(zap.Namespace("context")).Sugar().Fatalw(msg, keysAndValues...)
	} else {
		l.logger.Fatal(msg)
	}
}

// === Global Convenience Methods ===
// 等同于不带 context 调用的 ContextLogger 方法

func Debugw(msg string, keysAndValues ...interface{}) {
	Ctx(nil).Debugw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	Ctx(nil).Infow(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	Ctx(nil).Warnw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	Ctx(nil).Errorw(msg, keysAndValues...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	Ctx(nil).Fatalw(msg, keysAndValues...)
}
