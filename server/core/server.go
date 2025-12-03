package core

import (
	"log"
	"server/global"
	"server/initialize"
)

func RunServer() {
	serverCfg := global.CFG.Server

	r := initialize.NewRouter()
	if err := r.Run(serverCfg.Addr()); err != nil {
		log.Panicf("Run server error: %v\n", err)
	}
}
