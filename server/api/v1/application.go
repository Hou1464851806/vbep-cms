package v1

import (
	"github.com/gin-gonic/gin"
	"server/service"
)

type ApplicationApi struct {
	service *service.ApplicationService
}

func NewApplicationApi() *ApplicationApi {
	return &ApplicationApi{service: service.NewApplicationService()}
}

func (a *ApplicationApi) GetAll(ctx *gin.Context) {}

func (a *ApplicationApi) GetOne(ctx *gin.Context) {}

func (a *ApplicationApi) New(ctx *gin.Context) {}

func (a *ApplicationApi) Review(ctx *gin.Context) {}

func (a *ApplicationApi) Delete(ctx *gin.Context) {}
