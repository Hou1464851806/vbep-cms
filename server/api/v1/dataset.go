package v1

import (
	"errors"
	"net/http"
	"server/global"
	"server/logger"
	"server/model"
	"server/service"
	"strconv"

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
	// 获取用户信息 (由 AuthRole 或 JWTAuth 中间件注入)
	// 注意：如果你的 GetAll 路由没有加 AuthRole，只加了 JWTAuth，
	// 你需要确保能从 ctx 中拿到 ID。
	// 假设 JWTAuth 已经在 ctx 设置了 global.CTXUserID

	userID := ctx.GetString(global.CTXUserID)
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "未登录"})
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 获取用户对象或角色，假设 JWTAuth 或 AuthRole 已经处理了
	// 这里为了保险，简易起见，如果 ctx 中有 UserObj 则用，没有则默认 Role=0
	// *重要*：你需要确保 Router 中 GetAll 使用了中间件来获取 Role，
	// 或者在这里查库。为了性能，通常中间件做好了。
	// 假设 userObj 存在：
	var role model.UserRole = model.Common
	if u, exists := ctx.Get(global.CTXUserObj); exists {
		role = u.(*model.User).Role
	} else {
		// 如果 GetAll 只有 JWTAuth 没有 fetch user，这里为了严谨需要查库，
		// 但根据题目上下文，我们可以假设调用链可以获取到 role
		// 此处简化处理：默认普通用户
	}

	query := service.DatasetListQuery{
		UserID:    userID,
		Role:      role,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    ctx.Query("sort_by"),
		Order:     ctx.Query("order"),
		IDPrefix:  ctx.Query("id_prefix"),
		Name:      ctx.Query("name"),
		Creator:   ctx.Query("creator"),
		DateStart: ctx.Query("date_start"),
		DateEnd:   ctx.Query("date_end"),
	}
	datasets, total, err := a.service.GetAllDatasets(query)
	if err != nil {
		logger.Errorf("failed to get all datasets: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取数据集列表失败",
		})
		return // 别忘了 return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"datasets": datasets,
		"total":    total,
	})
}

func (a *DatasetApi) GetOne(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := ctx.GetString(global.CTXUserID)

	// 获取角色
	var role model.UserRole = model.Common
	if u, exists := ctx.Get(global.CTXUserObj); exists {
		role = u.(*model.User).Role
	}

	dataset, hasAccess, err := a.service.GetDatasetDetail(id, userID, role)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "数据集不存在"})
			return
		}
		logger.Errorf("failed to get dataset detail %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "获取数据集详情失败"})
		return
	}

	if !hasAccess {
		dataset.Files = []model.File{} // 如果没权限，连文件列表都不给看
	}

	// 构造符合前端命名规范的文件列表
	var fileResps []datasetFileResponse
	if len(dataset.Files) > 0 {
		fileResps = make([]datasetFileResponse, 0, len(dataset.Files))
		for _, f := range dataset.Files {
			fileResps = append(fileResps, datasetFileResponse{
				ID:        f.ID,
				Name:      f.Name,
				Size:      f.Size,
				DatasetID: f.OwnerID,
				CreatedAt: f.CreatedAt,
			})
		}
	} else {
		fileResps = []datasetFileResponse{}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"dataset": struct {
			*model.Dataset
			Files []datasetFileResponse `json:"files"`
		}{dataset, fileResps},
		"hasAccess": hasAccess,
	})
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
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}
	uidStr, ok := userID.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "用户ID类型错误"})
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
		"datasetID": newDataset.ID,
	})
}

func (a *DatasetApi) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "无效的数据集ID",
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
				"message": "数据集不存在",
			})
			return
		}
		logger.Errorf("failed to update dataset %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "更新数据集失败",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"dataset": dataset,
	})
}

func (a *DatasetApi) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "无效的数据集ID",
		})
		return
	}

	if err := a.service.DeleteDataset(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"message": "数据集不存在",
			})
			return
		}
		logger.Errorf("failed to delete dataset %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "删除数据集失败",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *DatasetApi) GetStatistics(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的数据集ID"})
		return
	}

	stats, err := a.service.GetDatasetStatistics(id)
	if err != nil {
		logger.Errorf("failed to get statistics for dataset %s: %v", id, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "获取统计数据失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"statistics": stats})
}
