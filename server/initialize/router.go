package initialize

import (
	"github.com/gin-gonic/gin"
	"net/http"
	api "server/api/v1"
	"server/global"
	"server/internal/templates"
	"server/router"
	"server/service"
	"strings"

	"go.uber.org/zap/zapcore"
	"server/logger"
)

func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	gin.DefaultWriter = logger.NewGinZapWriter(zapcore.InfoLevel)
	gin.DefaultErrorWriter = logger.NewGinZapWriter(zapcore.ErrorLevel)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(corsMiddleware())

	v1 := r.Group("/api/v1")

	v1.GET("/ping", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// email
	asynqClient := global.ASYNQ
	templateEngine := templates.NewTemplateEngine()
	mailService := service.NewMailService(asynqClient, templateEngine)

	// opt
	optService := service.NewOPTService(mailService)

	// user
	userService := service.NewUserServiceWithEmailService(mailService, optService)
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

	// license
	licenseService := service.NewLicenseService()
	licenseApi := api.NewLicenseApi(licenseService)
	licenseRouter := router.NewLicenseRouter(licenseApi)
	licenseRouter.InitLicenseRouter(v1)

	//mailApi := api.NewMailApi(mailService)
	//mailRouter := router.NewMailRouter(mailApi)
	//mailRouter.InitMailRouter(v1)

	return r
}

func corsMiddleware() gin.HandlerFunc {
	allowedOrigin := strings.TrimSpace(global.CFG.Web.Url)
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if allowedOrigin != "" && origin == allowedOrigin {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Credentials", "true")
			ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			ctx.Header("Vary", "Origin")
		}
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
