package service

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"server/global"
	"server/model"
)

type DatasetService struct {
}

func NewDatasetService() *DatasetService {
	return &DatasetService{}
}

func (s *DatasetService) CreateDataset(dataset *model.Dataset) error {
	ctx := context.Background()
	err := gorm.G[model.Dataset](global.DB).Create(ctx, dataset)
	if err != nil {
		return fmt.Errorf("failed to create new dataset: %w", err)
	}

	return nil
}

func (s *DatasetService) GetAllDatasets() ([]model.Dataset, error) {
	ctx := context.Background()
	var datasets []model.Dataset
	err := global.DB.WithContext(ctx).
		Preload("Files").
		Preload("Creator").
		Find(&datasets).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all datasets: %w", err)
	}

	return datasets, nil
}

func (s *DatasetService) GetDatasets() {

}

func (s *DatasetService) UpdateDataset(uuid string, title, description string) (*model.Dataset, error) {
	ctx := context.Background()
	updates := map[string]any{
		"title":       title,
		"description": description,
	}

	res := global.DB.WithContext(ctx).Model(&model.Dataset{}).Where("uuid = ?", uuid).Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("failed to update dataset: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	updated, err := gorm.G[model.Dataset](global.DB).Where("uuid = ?", uuid).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to query updated dataset: %w", err)
	}

	return &updated, nil
}

func (s *DatasetService) DeleteDataset(uuid string) error {
	ctx := context.Background()

	if err := NewFileService().DeleteFilesByDataset(uuid); err != nil {
		return fmt.Errorf("failed to delete files under dataset %s: %w", uuid, err)
	}

	res := global.DB.WithContext(ctx).Where("uuid = ?", uuid).Delete(&model.Dataset{})
	if res.Error != nil {
		return fmt.Errorf("failed to delete dataset: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
