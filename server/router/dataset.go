package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
)

type DatasetRouter struct {
}

func (r *DatasetRouter) InitDatasetRouter(router *gin.RouterGroup) {
	var datasetApi = v1.NewDatasetApi()

	datasetRouter := router.Group("/datasets")
	{
		datasetRouter.GET("/", datasetApi.GetAll)
		datasetRouter.GET("/:uuid", datasetApi.GetOne)
		datasetRouter.POST("/", datasetApi.New)
		datasetRouter.PATCH("/:uuid", datasetApi.Update)
		datasetRouter.DELETE("/:uuid", datasetApi.Delete)
	}
}
