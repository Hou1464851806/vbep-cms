package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"server/global"
	"server/logger"
	"server/model"
	"server/pkg/uuid"
)

const (
	maxUploadSize = 500 * 1024 * 1024
	publicOwnerID = "system"
)

const (
	datasetOwnerType     = "datasets"
	applicationOwnerType = "applications"
	publicOwnerType      = "public_downloads"
)

type FileService struct {
}

func NewFileService() *FileService {
	return &FileService{}
}

// --- Create ---

func (s *FileService) UploadFile(ctx context.Context, tx *gorm.DB, ownerID, ownerType, creatorID string, fileHeader *multipart.FileHeader) (*model.File, string, error) {
	if fileHeader == nil {
		return nil, "", fmt.Errorf("file header cannot be empty")
	}
	if fileHeader.Size > maxUploadSize {
		return nil, "", fmt.Errorf("file too large: maximum allowed size is %d MB", maxUploadSize/(1024*1024))
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, "", fmt.Errorf("failed to open upload file: %w", err)
	}
	defer src.Close()

	fileUUID := uuid.NewV4()
	ownerDir := model.GetDir(ownerType, ownerID)
	if err := os.MkdirAll(ownerDir, os.ModePerm); err != nil {
		return nil, "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	targetPath := filepath.Clean(filepath.Join(ownerDir, fileUUID))
	tmpFile, err := os.Create(targetPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer tmpFile.Close()

	hasher := md5.New()
	written, err := io.Copy(io.MultiWriter(tmpFile, hasher), src)
	if err != nil {
		// 清理可能残留的文件
		tmpFile.Close()
		os.Remove(targetPath)
		return nil, "", fmt.Errorf("failed to write file to disk: %w", err)
	}

	size := uint(written)
	newFile := &model.File{
		Base: model.Base{
			ID: fileUUID,
		},
		Name:      fileHeader.Filename,
		MD5:       hex.EncodeToString(hasher.Sum(nil)),
		Size:      size,
		OwnerID:   ownerID,
		OwnerType: ownerType,
		CreatorID: creatorID,
	}

	db := global.DB
	if tx != nil {
		db = tx
	}

	if err := db.WithContext(ctx).Create(newFile).Error; err != nil {
		_ = os.Remove(targetPath)
		return nil, "", fmt.Errorf("failed to save file metadata: %w", err)
	}

	return newFile, targetPath, nil
}

func (s *FileService) CreateDatasetFile(datasetID string, creatorID string, fileHeader *multipart.FileHeader) (*model.File, error) {
	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetID); err != nil {
		return nil, err
	}

	newFile, _, err := s.UploadFile(ctx, nil, datasetID, datasetOwnerType, creatorID, fileHeader)
	if err != nil {
		return nil, err
	}
	return newFile, nil
}

func (s *FileService) CreateApplicationFile(applicationID string, creatorID string, fileHeader *multipart.FileHeader) (*model.File, error) {
	ctx := context.Background()
	if err := s.ensureApplicationExists(ctx, applicationID); err != nil {
		return nil, err
	}

	newFile, _, err := s.UploadFile(ctx, nil, applicationID, applicationOwnerType, creatorID, fileHeader)
	if err != nil {
		return nil, err
	}
	return newFile, nil
}

func (s *FileService) CreatePublicFile(creatorID string, fileHeader *multipart.FileHeader) (*model.File, error) {
	ctx := context.Background()
	newFile, _, err := s.UploadFile(ctx, nil, publicOwnerID, publicOwnerType, creatorID, fileHeader)
	if err != nil {
		return nil, err
	}
	return newFile, nil
}

// --- Get ---

func (s *FileService) ListFiles(ctx context.Context, ownerID, ownerType string) ([]model.File, error) {
	var files []model.File
	err := global.DB.WithContext(ctx).
		Where("owner_id = ? AND owner_type = ?", ownerID, ownerType).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

func (s *FileService) GetFile(ctx context.Context, fileID, ownerID, ownerType string) (*model.File, error) {
	var file model.File
	err := global.DB.WithContext(ctx).
		Where("id = ? AND owner_id = ? AND owner_type = ?", fileID, ownerID, ownerType).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (s *FileService) GetFilesByDataset(datasetID string) ([]model.File, error) {
	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetID); err != nil {
		return nil, err
	}
	return s.ListFiles(ctx, datasetID, datasetOwnerType)
}

func (s *FileService) GetDatasetFile(datasetID, fileID string) (*model.File, error) {
	ctx := context.Background()
	file, err := s.GetFile(ctx, fileID, datasetID, datasetOwnerType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query dataset file: %w", err)
	}
	return file, nil
}

func (s *FileService) GetFilesByApplication(applicationID string) ([]model.File, error) {
	ctx := context.Background()
	if err := s.ensureApplicationExists(ctx, applicationID); err != nil {
		return nil, err
	}
	return s.ListFiles(ctx, applicationID, applicationOwnerType)
}

func (s *FileService) GetApplicationFile(applicationID, fileID string) (*model.File, error) {
	ctx := context.Background()
	file, err := s.GetFile(ctx, fileID, applicationID, applicationOwnerType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query application file: %w", err)
	}
	return file, nil
}

func (s *FileService) GetPublicFiles() ([]model.File, error) {
	ctx := context.Background()
	return s.ListFiles(ctx, publicOwnerID, publicOwnerType)
}

func (s *FileService) GetPublicFile(fileID string) (*model.File, error) {
	ctx := context.Background()
	file, err := s.GetFile(ctx, fileID, publicOwnerID, publicOwnerType)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query public file: %w", err)
	}
	return file, nil
}

func (s *FileService) GetPublicTemplateFiles() ([]model.File, string, error) {
	setting, err := GetSetting()
	if err != nil {
		return nil, "", err
	}
	if setting.TemplateFile == "" {
		return []model.File{}, setting.TemplateDescription, nil
	}

	ids := strings.Split(setting.TemplateFile, ",")
	var targetIDs []string
	for _, id := range ids {
		if t := strings.TrimSpace(id); t != "" {
			targetIDs = append(targetIDs, t)
		}
	}

	var files []model.File
	err = global.DB.WithContext(context.Background()).
		Where("id IN ? AND owner_id = ? AND owner_type = ?", targetIDs, publicOwnerID, publicOwnerType).
		Find(&files).Error

	return files, setting.TemplateDescription, err
}

// --- Delete ---

func (s *FileService) DeleteFile(ctx context.Context, fileID, ownerID, ownerType string) error {
	file, err := s.GetFile(ctx, fileID, ownerID, ownerType)
	if err != nil {
		return err
	}

	res := global.DB.WithContext(ctx).
		Where("id = ? AND owner_id = ? AND owner_type = ?", fileID, ownerID, ownerType).
		Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete file record: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	removePhysicalFile(file.GetPath())
	cleanupOwnerDirectory(ownerType, ownerID)
	return nil
}

func (s *FileService) DeleteDatasetFile(datasetID, fileID string) error {
	return s.DeleteFile(context.Background(), fileID, datasetID, datasetOwnerType)
}

func (s *FileService) DeleteFilesByDataset(datasetID string) error {
	ctx := context.Background()
	files, err := s.ListFiles(ctx, datasetID, datasetOwnerType)
	if err != nil {
		return fmt.Errorf("failed to query files for dataset %s: %w", datasetID, err)
	}

	for _, file := range files {
		removePhysicalFile(file.GetPath())
	}

	res := global.DB.WithContext(ctx).
		Where("owner_id = ? AND owner_type = ?", datasetID, datasetOwnerType).
		Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete files for dataset %s: %w", datasetID, res.Error)
	}

	if err := os.RemoveAll(model.GetDir(datasetOwnerType, datasetID)); err != nil && !os.IsNotExist(err) {
		logger.Warnf("failed to remove dataset directory %s: %v", datasetID, err)
	}
	return nil
}

func (s *FileService) DeleteApplicationFile(applicationID, fileID string) error {
	return s.DeleteFile(context.Background(), fileID, applicationID, applicationOwnerType)
}

func (s *FileService) DeletePublicFile(fileID string) error {
	return s.DeleteFile(context.Background(), fileID, publicOwnerID, publicOwnerType)
}

// --- Other ---

func (s *FileService) ensureDatasetExists(ctx context.Context, datasetID string) error {
	_, err := gorm.G[model.Dataset](global.DB).
		Select("id").
		Where("id = ?", datasetID).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		}
		return fmt.Errorf("failed to query dataset %s: %w", datasetID, err)
	}
	return nil
}

func (s *FileService) ensureApplicationExists(ctx context.Context, applicationID string) error {
	_, err := gorm.G[model.Application](global.DB).
		Select("id").
		Where("id = ?", applicationID).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		}
		return fmt.Errorf("failed to query application %s: %w", applicationID, err)
	}
	return nil
}

func removePhysicalFile(filePath string) {
	targetPath := filePath
	if !filepath.IsAbs(targetPath) {
		if abs, err := filepath.Abs(filePath); err == nil {
			targetPath = abs
		}
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		logger.Warnf("failed to remove file %s: %v", targetPath, err)
	}
}

func cleanupOwnerDirectory(ownerType, ownerID string) {
	dir := model.GetDir(ownerType, ownerID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warnf("failed to inspect directory %s: %v", dir, err)
		}
		return
	}
	if len(entries) == 0 {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			logger.Warnf("failed to remove directory %s: %v", dir, err)
		}
	}
}
