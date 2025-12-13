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
	datasetRouter.Use(middleware.JWTAuth())
	{
		datasetRouter.GET("/", middleware.AuthRole(model.Common), datasetApi.GetAll)
		datasetRouter.GET("/:id", middleware.AuthRole(model.Common), datasetApi.GetOne)
		datasetRouter.POST("/", middleware.AuthRole(model.Admin), datasetApi.New)
		datasetRouter.PATCH("/:id", middleware.AuthRole(model.Admin), datasetApi.Update)
		datasetRouter.DELETE("/:id", middleware.AuthRole(model.Admin), datasetApi.Delete)
	}
}
