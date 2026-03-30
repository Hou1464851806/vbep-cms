package v1

import (
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"server/global"
	"server/logger"
	"server/model"
	"server/service"
)

type FileApi struct {
	service            *service.FileService
	applicationService *service.ApplicationService
}

type datasetFileResponse struct {
	ID        string    `json:"fileID"`
	Name      string    `json:"name"`
	Size      uint      `json:"size"`
	DatasetID string    `json:"datasetID"`
	CreatedAt time.Time `json:"createdAt"`
}

type applicationFileResponse struct {
	ID            string    `json:"fileID"`
	Name          string    `json:"name"`
	Size          uint      `json:"size"`
	ApplicationID string    `json:"applicationID"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"createdAt"`
}

type publicFileResponse struct {
	ID        string    `json:"fileID"`
	Name      string    `json:"name"`
	Size      uint      `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewFileApi() *FileApi {
	return &FileApi{
		service:            service.NewFileService(),
		applicationService: service.NewApplicationService(),
	}
}

func (a *FileApi) ListByDataset(ctx *gin.Context) {
	datasetID := ctx.Param("id")
	if datasetID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的数据集ID"})
		return
	}

	files, err := a.service.GetFilesByDataset(datasetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "数据集不存在"})
			return
		}
		logger.Errorf("failed to list files for dataset %s: %v", datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载文件列表失败"})
		return
	}

	resp := make([]datasetFileResponse, 0, len(files))
	for _, file := range files {
		resp = append(resp, datasetFileResponse{
			ID:        file.ID,
			Name:      file.Name,
			Size:      file.Size,
			DatasetID: file.OwnerID,
			CreatedAt: file.CreatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"files": resp,
	})
}

func (a *FileApi) UploadToDataset(ctx *gin.Context) {
	datasetID := ctx.Param("id")
	if datasetID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的数据集ID"})
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "请上传文件"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	file, err := a.service.CreateDatasetFile(datasetID, user.ID, fileHeader)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "数据集不存在"})
			return
		}
		logger.Errorf("failed to upload file for dataset %s: %v", datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "保存文件失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"file": datasetFileResponse{
			ID:        file.ID,
			Name:      file.Name,
			Size:      file.Size,
			DatasetID: file.OwnerID,
			CreatedAt: file.CreatedAt,
		},
	})
}

func (a *FileApi) DownloadByDataset(ctx *gin.Context) {
	datasetID := ctx.Param("id")
	fileID := ctx.Param("fileID")
	if datasetID == "" || fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的数据集或文件ID"})
		return
	}

	// 获取当前用户
	userObj, exists := ctx.Get(global.CTXUserObj)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}
	user := userObj.(*model.User)

	// 执行权限检查
	if !a.checkDatasetDownloadPermission(datasetID, user.ID, user.Role) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "您没有权限下载此文件，请先申请并等待审批通过"})
		return
	}

	file, err := a.service.GetDatasetFile(datasetID, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
			return
		}
		logger.Errorf("failed to get file %s for dataset %s: %v", fileID, datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "下载失败"})
		return
	}

	targetPath := file.GetPath()
	if !filepath.IsAbs(targetPath) {
		if abs, err := filepath.Abs(targetPath); err == nil {
			targetPath = abs
		}
	}

	ctx.FileAttachment(targetPath, file.Name)
}

func (a *FileApi) DeleteFromDataset(ctx *gin.Context) {
	datasetID := ctx.Param("id")
	fileID := ctx.Param("fileID")
	if datasetID == "" || fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的数据集或文件ID"})
		return
	}

	if err := a.service.DeleteDatasetFile(datasetID, fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
			return
		}
		logger.Errorf("failed to delete file %s for dataset %s: %v", fileID, datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "删除文件失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *FileApi) ListByApplication(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	if applicationID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canAccessApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	files, err := a.service.GetLatestApplicationFiles(applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "申请不存在"})
			return
		}
		logger.Errorf("failed to list files for application %s: %v", applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载文件列表失败"})
		return
	}

	resp := make([]applicationFileResponse, 0, len(files))
	for _, file := range files {
		resp = append(resp, buildApplicationFileResponse(file))
	}

	ctx.JSON(http.StatusOK, gin.H{"files": resp})
}

func (a *FileApi) UploadToApplication(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	if applicationID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请ID"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canUploadApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "只有申请人可以在审批前上传附件"})
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "请上传文件"})
		return
	}

	fileType := ctx.PostForm("type")

	file, err := a.service.CreateApplicationFile(applicationID, user.ID, fileType, fileHeader)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "申请不存在"})
			return
		}
		logger.Errorf("failed to upload file for application %s: %v", applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "保存文件失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"file": buildApplicationFileResponse(*file)})
}

func (a *FileApi) DownloadByApplication(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	fileID := ctx.Param("fileID")
	if applicationID == "" || fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请或文件ID"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canAccessApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	file, err := a.service.GetApplicationFile(applicationID, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
			return
		}
		logger.Errorf("failed to get file %s for application %s: %v", fileID, applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "下载文件失败"})
		return
	}

	targetPath := file.GetPath()
	if !filepath.IsAbs(targetPath) {
		if abs, err := filepath.Abs(targetPath); err == nil {
			targetPath = abs
		}
	}

	ctx.FileAttachment(targetPath, file.Name)
}

func (a *FileApi) DeleteFromApplication(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	fileID := ctx.Param("fileID")
	if applicationID == "" || fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的申请或文件ID"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canDeleteApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	if err := a.service.DeleteApplicationFile(applicationID, fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
			return
		}
		logger.Errorf("failed to delete file %s for application %s: %v", fileID, applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "删除文件失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *FileApi) ListPublic(ctx *gin.Context) {
	if _, ok := getUserFromContext(ctx); !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
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

	query := service.PublicFileListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    ctx.Query("sort_by"),
		Order:     ctx.Query("order"),
		Keyword:   ctx.Query("keyword"),
		DateStart: ctx.Query("date_start"),
		DateEnd:   ctx.Query("date_end"),
	}
	files, total, err := a.service.GetPublicFilesPaginated(query)
	if err != nil {
		logger.Errorf("failed to list public files: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载文件列表失败"})
		return
	}

	resp := make([]publicFileResponse, 0, len(files))
	for _, file := range files {
		resp = append(resp, publicFileResponse{
			ID:        file.ID,
			Name:      file.Name,
			Size:      file.Size,
			CreatedAt: file.CreatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"files": resp,
		"total": total,
	})
}

func (a *FileApi) ListPublicTemplates(ctx *gin.Context) {
	if _, ok := getUserFromContext(ctx); !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	files, description, err := a.service.GetPublicTemplateFiles()
	if err != nil {
		logger.Errorf("failed to list public template files: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载模版文件失败"})
		return
	}

	resp := make([]publicFileResponse, 0, len(files))
	for _, file := range files {
		resp = append(resp, publicFileResponse{
			ID:        file.ID,
			Name:      file.Name,
			Size:      file.Size,
			CreatedAt: file.CreatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"files":       resp,
		"description": description,
	})
}

func (a *FileApi) UploadPublic(ctx *gin.Context) {
	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}
	if user.Role != model.SuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "请上传文件"})
		return
	}

	file, err := a.service.CreatePublicFile(user.ID, fileHeader)
	if err != nil {
		logger.Errorf("failed to upload public file: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "保存文件失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"file": publicFileResponse{
		ID:        file.ID,
		Name:      file.Name,
		Size:      file.Size,
		CreatedAt: file.CreatedAt,
	}})
}

func (a *FileApi) DownloadPublic(ctx *gin.Context) {
	fileID := ctx.Param("fileID")
	if fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的文件ID"})
		return
	}

	if _, ok := getUserFromContext(ctx); !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}

	file, err := a.service.GetPublicFile(fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
			return
		}
		logger.Errorf("failed to get public file %s: %v", fileID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "下载失败"})
		return
	}

	targetPath := file.GetPath()
	if !filepath.IsAbs(targetPath) {
		if abs, err := filepath.Abs(targetPath); err == nil {
			targetPath = abs
		}
	}

	ctx.FileAttachment(targetPath, file.Name)
}

func (a *FileApi) DeletePublic(ctx *gin.Context) {
	fileID := ctx.Param("fileID")
	if fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "无效的文件ID"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
		return
	}
	if user.Role != model.SuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "权限不足"})
		return
	}

	if err := a.service.DeletePublicFile(fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "文件不存在"})
			return
		}
		logger.Errorf("failed to delete public file %s: %v", fileID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "删除文件失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{})
}

func (a *FileApi) loadApplication(ctx *gin.Context, applicationID string) (*model.ApplicationDetail, bool) {
	application, err := a.applicationService.GetApplicationByID(applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "申请不存在"})
			return nil, false
		}
		logger.Errorf("failed to get application %s: %v", applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "加载申请失败"})
		return nil, false
	}
	return application, true
}

func getUserFromContext(ctx *gin.Context) (*model.User, bool) {
	userObj, exists := ctx.Get(global.CTXUserObj)
	if !exists {
		return nil, false
	}
	user, ok := userObj.(*model.User)
	return user, ok
}

func (a *FileApi) checkDatasetDownloadPermission(datasetID string, userID string, role model.UserRole) bool {
	// 管理员直接通过
	if role > model.Common {
		return true
	}

	// 普通用户：检查是否有 Approved 的申请
	var count int64
	err := global.DB.Model(&model.Application{}).
		Where("dataset_id = ? AND applicant_id = ? AND status = ?", datasetID, userID, model.APPROVED).
		Where("expire_at IS NULL OR expire_at > ?", time.Now()).
		Count(&count).Error

	if err != nil {
		logger.Errorf("failed to check permission: %v", err)
		return false
	}

	return count > 0
}

func canAccessApplicationFiles(user *model.User, application *model.ApplicationDetail) bool {
	if user == nil || application == nil {
		return false
	}
	if user.Role >= model.Admin {
		return true
	}
	return application.Applicant.ID == user.ID
}

func canUploadApplicationFiles(user *model.User, application *model.ApplicationDetail) bool {
	if user == nil || application == nil {
		return false
	}
	if application.Applicant.ID != user.ID {
		return false
	}
	return application.Status == "draft" || application.Status == "returned"
}

func canDeleteApplicationFiles(user *model.User, application *model.ApplicationDetail) bool {
	if user == nil || application == nil {
		return false
	}
	if user.Role >= model.Admin {
		return true
	}
	if application.Applicant.ID != user.ID {
		return false
	}
	return application.Status == "draft" || application.Status == "returned"
}

func buildApplicationFileResponse(file model.File) applicationFileResponse {
	return applicationFileResponse{
		ID:            file.ID,
		Name:          file.Name,
		Size:          file.Size,
		ApplicationID: file.OwnerID,
		Type:          file.Type,
		CreatedAt:     file.CreatedAt,
	}
}
