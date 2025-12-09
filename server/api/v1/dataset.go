package v1

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"server/logger"
	"server/model"
	"server/pkg/uuid"
	"server/service"
)

type DatasetApi struct {
	service *service.DatasetService
}

func NewDatasetApi() *DatasetApi {
	return &DatasetApi{service: service.NewDatasetService()}
}

func (a *DatasetApi) GetAll(ctx *gin.Context) {
	datasets, err := a.service.GetAllDatasets()
	if err != nil {
		logger.Errorf("failed to get all datasets: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":  "success",
		"datasets": datasets,
	})
}

func (a *DatasetApi) GetOne(ctx *gin.Context) {

}

type NewDatasetParam struct {
	UserID      uint   `json:"user_id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (a *DatasetApi) New(ctx *gin.Context) {
	var p NewDatasetParam
	if err := ctx.ShouldBind(&p); err != nil {
		logger.Errorf("failed to parse request param: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
	}

	err := a.service.CreateDataset(&model.Dataset{
		UUID:        uuid.NewV4(),
		Title:       p.Title,
		Description: p.Description,
		CreatedBy:   p.UserID,
	})
	if err != nil {
		logger.Errorf("%v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}

func (a *DatasetApi) Update(ctx *gin.Context) {

}

func (a *DatasetApi) Delete(ctx *gin.Context) {

}
