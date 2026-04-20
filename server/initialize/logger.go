package initialize

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"server/config"
)

const (
	ModeDebug = "debug"
	ModeInfo  = "info"
)

func NewLogger() (*zap.Logger, *zap.SugaredLogger) {

	cores := make([]zapcore.Core, 0)

	cores = append(cores, newFileZapCore())
	cores = append(cores, newConsoleZapCore())

	log := zap.New(zapcore.NewTee(cores...), zap.AddCallerSkip(1))
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

	c := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(f), zapcore.DebugLevel)
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
		name = "worker.log"
	default:
		name = "server.log"
	}
	return name
}

func newConsoleZapCore() zapcore.Core {
	mode := ModeInfo
	if m := os.Getenv("LOGGER_MODE"); m != "" {
		mode = m
	}
	var c zapcore.Core
	switch mode {
	case ModeDebug:
		c = zapcore.NewCore(getConsoleEncoder(), zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	case ModeInfo:
		c = zapcore.NewCore(getConsoleEncoder(), zapcore.AddSync(os.Stdout), zapcore.InfoLevel)
	default:
		c = zapcore.NewCore(getConsoleEncoder(), zapcore.AddSync(os.Stdout), zapcore.InfoLevel)
	}
	return c
}

func getConsoleEncoder() zapcore.Encoder {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05Z07:00")
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(config)
}
