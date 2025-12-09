package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
	"server/middleware"
)

type FileRouter struct {
}

func (r *FileRouter) InitFileRouter(router *gin.RouterGroup) {
	var fileApi = v1.NewFileApi()

	//fileRouter := router.Group("/files")

	router.GET("/datasets/files/:id", middleware.JWTAuth(), fileApi.DownloadByDataset)
	router.GET("/requests/files/:id", middleware.JWTAuth(), fileApi.DownloadByRequest)
	router.POST("/datasets/files/", middleware.JWTAuth(), fileApi.UploadToDataset)
	router.POST("/requests/files/", middleware.JWTAuth(), fileApi.UploadToRequest)

}
