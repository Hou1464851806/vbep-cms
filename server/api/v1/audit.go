package v1

import (
	"github.com/gin-gonic/gin"
	"server/service"
)

type AuditApi struct {
	service *service.AuditService
}

func NewAuditApi() *AuditApi {
	return &AuditApi{service: service.NewAuditService()}
}

func (a *AuditApi) GetOne(ctx *gin.Context) {}

func (a *AuditApi) GetAll(ctx *gin.Context) {}

func (a *AuditApi) NewOne(ctx *gin.Context) {}
