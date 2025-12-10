package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
	"server/middleware"
)

type DatasetRouter struct {
}

func (r *DatasetRouter) InitDatasetRouter(router *gin.RouterGroup) {
	var datasetApi = v1.NewDatasetApi()

	datasetRouter := router.Group("/datasets")
	{
		datasetRouter.GET("/", datasetApi.GetAll)
		datasetRouter.GET("/:datasetUUID", datasetApi.GetOne)
		datasetRouter.POST("/", middleware.JWTAuth(), datasetApi.New)
		datasetRouter.PATCH("/:datasetUUID", middleware.JWTAuth(), datasetApi.Update)
		datasetRouter.DELETE("/:datasetUUID", middleware.JWTAuth(), datasetApi.Delete)
	}
}
