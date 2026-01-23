package initialize

import (
	"net/http"
	"server/router"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/api/v1")

	v1.GET("/ping", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// user
	userRouter := new(router.UserRouter)
	userRouter.InitUserRouter(v1)

	// dataset
	datasetRouter := new(router.DatasetRouter)
	datasetRouter.InitDatasetRouter(v1)

	// file
	fileRouter := new(router.FileRouter)
	fileRouter.InitFileRouter(v1)

	// application
	applicationRouter := new(router.ApplicationRouter)
	applicationRouter.InitApplicationRouter(v1)

	// setting
	settingRouter := new(router.SettingRouter)
	settingRouter.InitSettingRouter(v1)

	return r
}
