package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type FileRouter struct {
}

func (r *FileRouter) InitFileRouter(router *gin.RouterGroup) {
	var fileApi = v1.NewFileApi()

	datasetFiles := router.Group("/datasets/:id/files")
	datasetFiles.Use(middleware.JWTAuth())
	{
		datasetFiles.GET("", middleware.JWTAuth(), middleware.AuthRole(model.Common), fileApi.ListByDataset)
		datasetFiles.POST("", middleware.JWTAuth(), middleware.AuthRole(model.Admin), fileApi.UploadToDataset)
		datasetFiles.GET("/:fileID", middleware.JWTAuth(), middleware.AuthRole(model.Common), fileApi.DownloadByDataset)
		datasetFiles.DELETE("/:fileID", middleware.JWTAuth(), middleware.AuthRole(model.Admin), fileApi.DeleteFromDataset)
	}
}
