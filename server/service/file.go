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

	"gorm.io/gorm"

	"server/global"
	"server/logger"
	"server/model"
	"server/pkg/uuid"
)

const (
	datasetUploadRoot = "uploads/datasets"
	maxUploadSize     = 50 * 1024 * 1024

	// Default GORM table name for Dataset is "datasets", which is stored in OwnerType.
	datasetOwnerType = "datasets"
)

type FileService struct {
}

func NewFileService() *FileService {
	return &FileService{}
}

func (s *FileService) CreateDatasetFile(datasetID string, fileHeader *multipart.FileHeader) (*model.File, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("file header cannot be empty")
	}
	if fileHeader.Size > maxUploadSize {
		return nil, fmt.Errorf("file too large: maximum allowed size is %d MB", maxUploadSize/(1024*1024))
	}

	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetID); err != nil {
		return nil, err
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open upload file: %w", err)
	}
	defer src.Close()

	fileUUID := uuid.NewV4()
	datasetDir := datasetDirPath(datasetID)
	if err := os.MkdirAll(datasetDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create dataset upload directory: %w", err)
	}

	fileExt := filepath.Ext(fileHeader.Filename)
	targetPath := filepath.Clean(filepath.Join(datasetDir, fmt.Sprintf("%s%s", fileUUID, fileExt)))
	tmpFile, err := os.Create(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer tmpFile.Close()

	hasher := md5.New()
	written, err := io.Copy(io.MultiWriter(tmpFile, hasher), src)
	if err != nil {
		return nil, fmt.Errorf("failed to write file to disk: %w", err)
	}

	size := uint(written)
	newFile := &model.File{
		Base: model.Base{
			ID: fileUUID,
		},
		Name:      fileHeader.Filename,
		Path:      filepath.ToSlash(targetPath),
		MD5:       hex.EncodeToString(hasher.Sum(nil)),
		Size:      size,
		OwnerID:   datasetID,
		OwnerType: datasetOwnerType,
	}

	if err := gorm.G[model.File](global.DB).Create(ctx, newFile); err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	return newFile, nil
}

func (s *FileService) GetFilesByDataset(datasetID string) ([]model.File, error) {
	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetID); err != nil {
		return nil, err
	}

	files, err := gorm.G[model.File](global.DB).
		Where("owner_id = ? AND owner_type = ?", datasetID, datasetOwnerType).
		Order("created_at DESC").
		Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query files for dataset: %w", err)
	}

	return files, nil
}

func (s *FileService) GetDatasetFile(datasetID, fileID string) (*model.File, error) {
	ctx := context.Background()
	file, err := gorm.G[model.File](global.DB).
		Where("id = ? AND owner_id = ? AND owner_type = ?", fileID, datasetID, datasetOwnerType).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query dataset file: %w", err)
	}
	return &file, nil
}

func (s *FileService) DeleteDatasetFile(datasetID, fileID string) error {
	file, err := s.GetDatasetFile(datasetID, fileID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	res := global.DB.WithContext(ctx).
		Where("id = ? AND owner_id = ? AND owner_type = ?", fileID, datasetID, datasetOwnerType).
		Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete file record: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	removePhysicalFile(file.Path)
	cleanupDatasetDirectory(datasetID)
	return nil
}

func (s *FileService) DeleteFilesByDataset(datasetID string) error {
	ctx := context.Background()
	files, err := gorm.G[model.File](global.DB).
		Where("owner_id = ? AND owner_type = ?", datasetID, datasetOwnerType).
		Find(ctx)
	if err != nil {
		return fmt.Errorf("failed to query files for dataset %s: %w", datasetID, err)
	}

	for _, file := range files {
		removePhysicalFile(file.Path)
	}

	res := global.DB.WithContext(ctx).
		Where("owner_id = ? AND owner_type = ?", datasetID, datasetOwnerType).
		Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete files for dataset %s: %w", datasetID, res.Error)
	}

	if err := os.RemoveAll(datasetDirPath(datasetID)); err != nil && !os.IsNotExist(err) {
		logger.Warnf("failed to remove dataset directory %s: %v", datasetID, err)
	}
	return nil
}

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

func datasetDirPath(datasetID string) string {
	return filepath.Join(datasetUploadRoot, datasetID)
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

func cleanupDatasetDirectory(datasetUUID string) {
	dir := datasetDirPath(datasetUUID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warnf("failed to inspect dataset directory %s: %v", dir, err)
		}
		return
	}
	if len(entries) == 0 {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			logger.Warnf("failed to remove dataset directory %s: %v", dir, err)
		}
	}
}
