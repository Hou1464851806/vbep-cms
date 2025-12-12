package v1

import (
	"errors"
	"net/http"
	"path/filepath"
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
	CreatedAt     time.Time `json:"createdAt"`
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
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid dataset id"})
		return
	}

	files, err := a.service.GetFilesByDataset(datasetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "dataset not found"})
			return
		}
		logger.Errorf("failed to list files for dataset %s: %v", datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load files"})
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
		"message": "success",
		"files":   resp,
	})
}

func (a *FileApi) UploadToDataset(ctx *gin.Context) {
	datasetID := ctx.Param("id")
	if datasetID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid dataset id"})
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "file is required"})
		return
	}

	file, err := a.service.CreateDatasetFile(datasetID, fileHeader)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "dataset not found"})
			return
		}
		logger.Errorf("failed to upload file for dataset %s: %v", datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save file"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "success",
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
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid dataset or file id"})
		return
	}

	file, err := a.service.GetDatasetFile(datasetID, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
			return
		}
		logger.Errorf("failed to get file %s for dataset %s: %v", fileID, datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to download file"})
		return
	}

	targetPath := file.Path
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
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid dataset or file id"})
		return
	}

	if err := a.service.DeleteDatasetFile(datasetID, fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
			return
		}
		logger.Errorf("failed to delete file %s for dataset %s: %v", fileID, datasetID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete file"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (a *FileApi) ListByApplication(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	if applicationID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application id"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "user not authenticated"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canAccessApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "permission denied"})
		return
	}

	files, err := a.service.GetFilesByApplication(applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "application not found"})
			return
		}
		logger.Errorf("failed to list files for application %s: %v", applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load files"})
		return
	}

	resp := make([]applicationFileResponse, 0, len(files))
	for _, file := range files {
		resp = append(resp, buildApplicationFileResponse(file))
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success", "files": resp})
}

func (a *FileApi) UploadToApplication(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	if applicationID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application id"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "user not authenticated"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canUploadApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "only applicant can upload attachments before approval"})
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "file is required"})
		return
	}

	file, err := a.service.CreateApplicationFile(applicationID, fileHeader)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "application not found"})
			return
		}
		logger.Errorf("failed to upload file for application %s: %v", applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save file"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success", "file": buildApplicationFileResponse(*file)})
}

func (a *FileApi) DownloadByApplication(ctx *gin.Context) {
	applicationID := ctx.Param("id")
	fileID := ctx.Param("fileID")
	if applicationID == "" || fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application or file id"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "user not authenticated"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canAccessApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "permission denied"})
		return
	}

	file, err := a.service.GetApplicationFile(applicationID, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
			return
		}
		logger.Errorf("failed to get file %s for application %s: %v", fileID, applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to download file"})
		return
	}

	targetPath := file.Path
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
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid application or file id"})
		return
	}

	user, ok := getUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "user not authenticated"})
		return
	}

	application, ok := a.loadApplication(ctx, applicationID)
	if !ok {
		return
	}

	if !canDeleteApplicationFiles(user, application) {
		ctx.JSON(http.StatusForbidden, gin.H{"message": "permission denied"})
		return
	}

	if err := a.service.DeleteApplicationFile(applicationID, fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
			return
		}
		logger.Errorf("failed to delete file %s for application %s: %v", fileID, applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete file"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (a *FileApi) loadApplication(ctx *gin.Context, applicationID string) (*model.ApplicationListItem, bool) {
	application, err := a.applicationService.GetApplicationByID(applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"message": "application not found"})
			return nil, false
		}
		logger.Errorf("failed to get application %s: %v", applicationID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load application"})
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

func canAccessApplicationFiles(user *model.User, application *model.ApplicationListItem) bool {
	if user == nil || application == nil {
		return false
	}
	if user.Role >= model.Admin {
		return true
	}
	return application.Applicant.ID == user.ID
}

func canUploadApplicationFiles(user *model.User, application *model.ApplicationListItem) bool {
	if user == nil || application == nil {
		return false
	}
	return application.Applicant.ID == user.ID && application.Status == "pending"
}

func canDeleteApplicationFiles(user *model.User, application *model.ApplicationListItem) bool {
	if user == nil || application == nil {
		return false
	}
	if user.Role >= model.Admin {
		return true
	}
	return application.Applicant.ID == user.ID && application.Status == "pending"
}

func buildApplicationFileResponse(file model.File) applicationFileResponse {
	return applicationFileResponse{
		ID:            file.ID,
		Name:          file.Name,
		Size:          file.Size,
		ApplicationID: file.OwnerID,
		CreatedAt:     file.CreatedAt,
	}
}
