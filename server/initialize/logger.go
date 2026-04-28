package initialize

import (
	"fmt"
	"os"
	"server/config"
	"server/pkg/utils"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ModeDebug = "debug"
	ModeInfo  = "info"
)

func NewLogger() (*zap.Logger, *zap.SugaredLogger) {
	serviceName := "vbep-cms-api"
	mode := config.AppModeApi
	if m := os.Getenv("APP_MODE"); m != "" {
		mode = m
	}
	if mode == config.AppModeWorker {
		serviceName = "vbep-cms-worker"
	}

	cores := make([]zapcore.Core, 0)

	// Prod环境下写入文件
	if utils.IsProduction() {
		cores = append(cores, newFileZapCore())
	}
	// Dev和Prod环境下都输出到控制台
	cores = append(cores, newConsoleZapCore())

	// 移除全局的 AddCallerSkip，后续在封装的 ContextLogger 中进行控制
	log := zap.New(zapcore.NewTee(cores...), zap.AddCaller())
	// 无论Dev还是Prod都自动填好 service
	log = log.With(zap.String("service", serviceName))

	return log, log.Sugar()
}

func newFileZapCore() zapcore.Core {
	err := os.Mkdir("logs", 0755)
	if err != nil && !os.IsExist(err) {
		panic(fmt.Sprintf("create log directory error: %v\n", err))
	}

	logPath := fmt.Sprintf("logs/%s", resolveLogFileName())
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(fmt.Sprintf("open log file error: %v\n", err))
	}

	c := zapcore.NewCore(getJSONEncoder(), zapcore.AddSync(f), zapcore.DebugLevel)
	return c
}

func resolveLogFileName() string {
	mode := config.AppModeApi
	if m := os.Getenv("APP_MODE"); m != "" {
		mode = m
	}
	var name string
	switch mode {
	case config.AppModeWorker:
		name = "worker"
	default:
		name = "server"
	}
	return fmt.Sprintf("%s-%s.log", name, time.Now().Format("2006-01-02"))
}

func newConsoleZapCore() zapcore.Core {
	mode := ModeInfo
	if m := os.Getenv("LOGGER_MODE"); m != "" {
		mode = m
	}

	// 统一使用 JSON Encoder
	encoder := getJSONEncoder()

	var c zapcore.Core
	switch mode {
	case ModeDebug:
		c = zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	case ModeInfo:
		c = zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel)
	default:
		c = zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.InfoLevel)
	}
	return c
}

func getJSONEncoder() zapcore.Encoder {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.LevelKey = "level"
	config.MessageKey = "message"
	config.CallerKey = "caller"
	config.StacktraceKey = "stacktrace"
	config.EncodeTime = zapcore.EpochTimeEncoder // Float秒级时间戳: 1714016402.123
	config.EncodeLevel = zapcore.LowercaseLevelEncoder
	return zapcore.NewJSONEncoder(config)
}
