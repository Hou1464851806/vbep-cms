package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"server/config"
	"server/logger"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ModeDebug = "debug"
	ModeInfo  = "info"
	ModeWarn  = "warn"
	ModeError = "error"
)

func NewLogger() (*zap.Logger, *zap.SugaredLogger) {
	cores := make([]zapcore.Core, 0)
	loggerCfg := resolveLogLevelConfig()

	cores = append(cores, newFileZapCore(loggerCfg))
	cores = append(cores, newConsoleZapCore(loggerCfg))

	log := zap.New(zapcore.NewTee(cores...))
	log = log.With(zap.String("service", resolveServiceName()))
	return log, log.Sugar()
}

func newFileZapCore(loggerCfg config.LoggerConfig) zapcore.Core {
	err := os.Mkdir("logs", 0755)
	if err != nil && !os.IsExist(err) {
		panic(fmt.Sprintf("create log directory error: %v\n", err))
	}

	logPath := filepath.Join("logs", resolveLogFileName())
	fileCfg := resolveLogFileConfig()
	fileWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    fileCfg.MaxSize,
		MaxBackups: fileCfg.MaxBackups,
		MaxAge:     fileCfg.MaxAge,
		Compress:   fileCfg.Compress,
	}

	return logger.NewStructuredCore(
		zapcore.NewCore(zapcore.NewJSONEncoder(getFileEncoderConfig()), zapcore.AddSync(fileWriter), resolveLoggerLevel(loggerCfg.FileLevel, zapcore.DebugLevel)),
		logger.OutputModeJSON,
	)
}

func resolveLogLevelConfig() config.LoggerConfig {
	var loggerCfg config.LoggerConfig
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err == nil {
		_ = v.UnmarshalKey("logger", &loggerCfg)
	}
	if consoleLevel := os.Getenv("LOGGER_CONSOLE_LEVEL"); consoleLevel != "" {
		loggerCfg.ConsoleLevel = consoleLevel
	}
	if fileLevel := os.Getenv("LOGGER_FILE_LEVEL"); fileLevel != "" {
		loggerCfg.FileLevel = fileLevel
	}
	return loggerCfg
}

func resolveLoggerLevel(level string, fallback zapcore.Level) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case ModeDebug:
		return zapcore.DebugLevel
	case ModeInfo:
		return zapcore.InfoLevel
	case ModeWarn:
		return zapcore.WarnLevel
	case ModeError:
		return zapcore.ErrorLevel
	default:
		return fallback
	}
}

func resolveLogFileConfig() config.LoggerFileConfig {
	var fileCfg config.LoggerFileConfig
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err == nil {
		_ = v.UnmarshalKey("logger.file", &fileCfg)
	}
	if fileCfg.MaxSize <= 0 {
		fileCfg.MaxSize = 100
	}
	if fileCfg.MaxBackups <= 0 {
		fileCfg.MaxBackups = 10
	}
	if fileCfg.MaxAge <= 0 {
		fileCfg.MaxAge = 30
	}
	return fileCfg
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

func newConsoleZapCore(loggerCfg config.LoggerConfig) zapcore.Core {
	return logger.NewStructuredCore(
		zapcore.NewCore(getConsoleEncoder(), zapcore.AddSync(os.Stdout), resolveLoggerLevel(loggerCfg.ConsoleLevel, zapcore.InfoLevel)),
		logger.OutputModeConsole,
	)
}

func getFileEncoderConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.CallerKey = "caller"
	config.FunctionKey = ""
	config.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339Nano)
	config.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(caller.File)
	}
	return config
}

func getJSONEncoder() zapcore.Encoder {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.CallerKey = "caller"
	config.FunctionKey = ""
	config.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05Z07:00")
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(caller.File)
	}
	return zapcore.NewConsoleEncoder(config)
}

func resolveServiceName() string {
	mode := config.AppModeApi
	if m := os.Getenv("APP_MODE"); m != "" {
		mode = m
	}
	return mode
}
