package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"server/global"
	"server/initialize"
)

func main() {
	Initialize()
	RunServer()
}

func Initialize() {
	global.VP = initialize.NewViper()
}

func RunServer() {
	r := gin.Default()
	r.GET("/ping", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	serverCfg := global.CFG.Server
	if err := r.Run(serverCfg.Addr()); err != nil {
		log.Panicf("Run server error: %v\n", err)
	}
}
