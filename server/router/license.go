package router

import (
	v1 "server/api/v1"
	"server/middleware"
	"server/model"

	"github.com/gin-gonic/gin"
)

type LicenseRouter struct {
	api *v1.LicenseApi
}

func NewLicenseRouter(api *v1.LicenseApi) *LicenseRouter {
	return &LicenseRouter{
		api: api,
	}
}

func (r *LicenseRouter) InitLicenseRouter(router *gin.RouterGroup) {
	licenseApi := r.api

	licenseRouter := router.Group("/licenses")
	licenseRouter.Use(middleware.JWTAuth())
	{
		licenseRouter.GET("/", middleware.AuthRole(model.Common), licenseApi.GetAll)
		licenseRouter.PUT("/:id/revoke", middleware.AuthRole(model.Admin), licenseApi.Revoke)
	}
}
