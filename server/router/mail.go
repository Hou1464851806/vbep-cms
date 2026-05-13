package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
)

type MailRouter struct {
	api *v1.MailApi
}

func NewMailRouter(api *v1.MailApi) *MailRouter {
	return &MailRouter{
		api: api,
	}
}

func (r *MailRouter) InitMailRouter(router *gin.RouterGroup) {
	mailRouter := router.Group("/email")
	{
		mailRouter.POST("/resetPassword", r.api.SendResetPassword)
		mailRouter.POST("/notification", r.api.SendNotification)
	}
}
