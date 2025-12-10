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

	datasetFiles := router.Group("/datasets/:datasetUUID/files")
	datasetFiles.Use(middleware.JWTAuth())
	{
		datasetFiles.GET("", fileApi.ListByDataset)
		datasetFiles.POST("", fileApi.UploadToDataset)
		datasetFiles.GET("/:fileUUID", fileApi.DownloadByDataset)
		datasetFiles.DELETE("/:fileUUID", fileApi.DeleteFromDataset)
	}
}
