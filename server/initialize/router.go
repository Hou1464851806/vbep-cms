package initialize

import (
	"github.com/gin-gonic/gin"
	"net/http"
	api "server/api/v1"
	"server/global"
	"server/router"
	"server/service"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/api/v1")

	v1.GET("/ping", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// email
	asynqClient := global.ASYNQ
	mailService := service.NewMailService(asynqClient)

	// user
	userService := service.NewUserServiceWithEmailService(mailService)
	userApi := api.NewUserApi(userService)
	userRouter := router.NewUserRouter(userApi)
	userRouter.InitUserRouter(v1)

	// dataset
	datasetRouter := new(router.DatasetRouter)
	datasetRouter.InitDatasetRouter(v1)

	// file
	fileRouter := new(router.FileRouter)
	fileRouter.InitFileRouter(v1)

	// application
	applicationService := service.NewApplicationServiceWithOther(mailService, service.NewSettingService())
	applicationApi := api.NewApplicationApi(applicationService)
	applicationRouter := router.NewApplicationRouter(applicationApi)
	applicationRouter.InitApplicationRouter(v1)

	// setting
	settingRouter := new(router.SettingRouter)
	settingRouter.InitSettingRouter(v1)

	// statistic
	statisticRouter := new(router.StatisticRouter)
	statisticRouter.InitStatisticRouter(v1)

	//mailApi := api.NewMailApi(mailService)
	//mailRouter := router.NewMailRouter(mailApi)
	//mailRouter.InitMailRouter(v1)

	return r
}
