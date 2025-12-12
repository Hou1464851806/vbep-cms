package v1

import (
	"errors"
	"net/http"
	"server/logger"
	"server/model"
	"server/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
			"message": "获取数据集列表失败", // 生产环境建议模糊错误信息
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
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type UpdateDatasetParam struct {
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
		return
	}

	// 从 JWT 获取 userID
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "user not authenticated"})
		return
	}
	uidStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "user id type error"})
		return
	}

	newDataset := &model.Dataset{
		Title:       p.Title,
		Description: p.Description,
		CreatorID:   uidStr,
	}
	err := a.service.CreateDataset(newDataset)
	if err != nil {
		logger.Errorf("%v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"id":      newDataset.ID,
	})
}

func (a *DatasetApi) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid dataset id",
		})
		return
	}

	var p UpdateDatasetParam
	if err := ctx.ShouldBind(&p); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	dataset, err := a.service.UpdateDataset(id, p.Title, p.Description)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "dataset not found",
			})
			return
		}
		logger.Errorf("failed to update dataset %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to update dataset",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
		"dataset": dataset,
	})
}

func (a *DatasetApi) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid dataset id",
		})
		return
	}

	if err := a.service.DeleteDataset(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "dataset not found",
			})
			return
		}
		logger.Errorf("failed to delete dataset %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to delete dataset",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}
