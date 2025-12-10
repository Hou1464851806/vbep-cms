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

const datasetUploadRoot = "uploads/datasets"

type FileService struct {
}

func NewFileService() *FileService {
	return &FileService{}
}

func (s *FileService) CreateDatasetFile(datasetUUID string, fileHeader *multipart.FileHeader) (*model.File, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("file header cannot be empty")
	}

	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetUUID); err != nil {
		return nil, err
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open upload file: %w", err)
	}
	defer src.Close()

	fileUUID := uuid.NewV4()
	datasetDir := filepath.Join(datasetUploadRoot, datasetUUID)
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
		UUID:        fileUUID,
		Name:        fileHeader.Filename,
		Path:        filepath.ToSlash(targetPath),
		MD5:         hex.EncodeToString(hasher.Sum(nil)),
		Size:        size,
		DatasetUUID: datasetUUID,
	}

	if err := gorm.G[model.File](global.DB).Create(ctx, newFile); err != nil {
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}

	return newFile, nil
}

func (s *FileService) GetFilesByDataset(datasetUUID string) ([]model.File, error) {
	ctx := context.Background()
	if err := s.ensureDatasetExists(ctx, datasetUUID); err != nil {
		return nil, err
	}

	files, err := gorm.G[model.File](global.DB).
		Where("dataset_uuid = ?", datasetUUID).
		Order("created_at DESC").
		Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query files for dataset: %w", err)
	}

	return files, nil
}

func (s *FileService) GetDatasetFile(datasetUUID, fileUUID string) (*model.File, error) {
	ctx := context.Background()
	file, err := gorm.G[model.File](global.DB).
		Where("uuid = ? AND dataset_uuid = ?", fileUUID, datasetUUID).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query dataset file: %w", err)
	}
	return &file, nil
}

func (s *FileService) DeleteDatasetFile(datasetUUID, fileUUID string) error {
	file, err := s.GetDatasetFile(datasetUUID, fileUUID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	res := global.DB.WithContext(ctx).Where("uuid = ?", fileUUID).Delete(&model.File{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete file record: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	absPath, err := filepath.Abs(file.Path)
	if err != nil {
		logger.Warnf("failed to compute absolute path for %s: %v", file.Path, err)
		return nil
	}
	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		logger.Warnf("failed to remove file %s: %v", absPath, err)
	}
	return nil
}

func (s *FileService) ensureDatasetExists(ctx context.Context, datasetUUID string) error {
	_, err := gorm.G[model.Dataset](global.DB).
		Select("uuid").
		Where("uuid = ?", datasetUUID).
		First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		}
		return fmt.Errorf("failed to query dataset %s: %w", datasetUUID, err)
	}
	return nil
}
