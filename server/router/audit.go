package router

import (
	"github.com/gin-gonic/gin"
	v1 "server/api/v1"
)

type AuditRouter struct {
}

func (r *AuditRouter) InitAuditRouter(router *gin.RouterGroup) {
	var auditApi = v1.NewAuditApi()

	audit := router.Group("/datasets/:id/audits")
	{
		audit.GET("/", auditApi.GetAll)
		audit.POST("/", auditApi.NewOne)
		audit.GET("/:auditID", auditApi.GetOne)
	}
}
