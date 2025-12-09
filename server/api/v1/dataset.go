package v1

import (
	"github.com/gin-gonic/gin"
	"server/service"
)

type DatasetApi struct {
	service *service.DatasetService
}

func NewDatasetApi() *DatasetApi {
	return &DatasetApi{service: service.NewDatasetService()}
}

func (a *DatasetApi) GetAll(ctx *gin.Context) {

}

func (a *DatasetApi) GetOne(ctx *gin.Context) {

}

func (a *DatasetApi) New(ctx *gin.Context) {

}

func (a *DatasetApi) Update(ctx *gin.Context) {

}

func (a *DatasetApi) Delete(ctx *gin.Context) {

}
