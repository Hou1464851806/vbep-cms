package initialize

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
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
	f, err := os.OpenFile("logs/server.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(fmt.Sprintf("open log file error: %v\n", err))
	}

	c := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(f), zapcore.DebugLevel)
	return c
}

func newConsoleZapCore() zapcore.Core {
	c := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
	return c
}
