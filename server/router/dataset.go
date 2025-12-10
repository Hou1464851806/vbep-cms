package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type DatasetRouter struct {
}

func (r *DatasetRouter) InitDatasetRouter(router *gin.RouterGroup) {
	var datasetApi = v1.NewDatasetApi()

	datasetRouter := router.Group("/datasets")
	{
		datasetRouter.GET("/", datasetApi.GetAll)
		datasetRouter.GET("/:id", datasetApi.GetOne)
		datasetRouter.POST("/", middleware.JWTAuth(), middleware.AuthRole(model.Admin), datasetApi.New)
		datasetRouter.PATCH("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Admin), datasetApi.Update)
		datasetRouter.DELETE("/:id", middleware.JWTAuth(), middleware.AuthRole(model.Admin), datasetApi.Delete)
	}
}
