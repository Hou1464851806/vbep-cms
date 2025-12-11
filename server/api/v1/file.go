package v1

import (
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"server/logger"
	"server/service"
)

type FileApi struct {
	service *service.FileService
}

type datasetFileResponse struct {
	ID        string    `json:"fileID"`
	Name      string    `json:"name"`
	Size      uint      `json:"size"`
	DatasetID string    `json:"datasetID"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewFileApi() *FileApi {
	return &FileApi{service: service.NewFileService()}
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
