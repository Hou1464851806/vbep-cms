package router

import (
	v1 "server/api/v1"
	"server/middleware"

	"github.com/gin-gonic/gin"
)

type FileRouter struct {
}

func (r *FileRouter) InitFileRouter(router *gin.RouterGroup) {
	var fileApi = v1.NewFileApi()

	datasetFiles := router.Group("/datasets/:id/files")
	datasetFiles.Use(middleware.JWTAuth())
	{
		datasetFiles.GET("", fileApi.ListByDataset)
		datasetFiles.POST("", fileApi.UploadToDataset)
		datasetFiles.GET("/:fileID", fileApi.DownloadByDataset)
		datasetFiles.DELETE("/:fileID", fileApi.DeleteFromDataset)
	}
}
