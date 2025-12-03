package main

import (
	"server/core"
	"server/global"
	"server/initialize"
)

func main() {
	initializeServer()
	core.RunServer()
}

func initializeServer() {
	global.LOG, global.SLOG = initialize.NewLogger()
	global.VP = initialize.NewViper()
	global.DB = initialize.NewDB()
}
