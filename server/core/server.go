package core

import (
	"server/global"
	"server/initialize"
	"server/logger"

	"go.uber.org/zap"
)

func RunServer() {
	serverCfg := global.CFG.Server

	r := initialize.NewRouter()
	if err := r.Run(serverCfg.Addr()); err != nil {
		logger.Fatal("run server failed", zap.Error(err), zap.String("addr", serverCfg.Addr()))
	}
}
